package proxy

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/transport"
)

// New creates a new proxy instance.
func New(dep Dependencies, opts Options) Proxy {
	if opts.QueueSize <= 0 {
		opts.QueueSize = 100
	}
	if opts.QueueTTL <= 0 {
		opts.QueueTTL = 30 * time.Second
	}

	return &proxy{
		state:            StateStopped,
		queue:            NewQueue(opts.QueueSize, opts.QueueTTL),
		logger:           dep.Logger,
		sanitizer:        dep.Sanitizer,
		supervisor:       dep.Supervisor,
		child:            dep.Child,
		client:           dep.Client,
		recorder:         dep.Recorder,
		redactor:         dep.Redactor,
		stateStore:       dep.StateStore,
		metricsCollector: dep.MetricsCollector,
		learner:          dep.AdaptiveLearner,
		hydraTools: func() []injectable.ToolDefinition {
			if len(opts.HydraTools) > 0 {
				return opts.HydraTools
			}
			return dep.HydraTools
		}(),
		collisionPolicy: opts.CollisionPolicy,
		crashExportPath: func() string {
			if opts.CrashExportPath != "" {
				return opts.CrashExportPath
			}
			return "/tmp/hydra-recorder-crash.json"
		}(),
		crashExportEnabled: opts.CrashExportEnabled,
		maxRestartsInWnd:   dep.MaxRestartsInWindow,
		maxRestarts:        dep.MaxRestarts,
		stopCh:             make(chan struct{}),
	}
}

// Run starts the proxy and begins processing messages.
func (p *proxy) Run() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if p.logger != nil {
				p.logger.Error("proxy panic", fmt.Errorf("%v", r))
			}
			p.sendErrorResponse(nil, "Internal server error")
			err = nil
		}
	}()

	p.started = time.Now()
	if p.supervisor != nil {
		p.state = StateStarting
		if err := p.supervisor.Start(); err != nil {
			p.state = StateFailed
			p.lastError = err.Error()
			return fmt.Errorf("failed to start supervisor: %w", err)
		}
	}

	if p.mainLoop != nil {
		return p.mainLoop()
	}

	p.state = StateRunning
	return p.loop()
}

// Shutdown stops the proxy and cleans up resources.
func (p *proxy) Shutdown() error {
	p.state = StateStopped
	if p.child != nil {
		_ = p.child.Close()
	}
	if p.supervisor != nil {
		return p.supervisor.Stop()
	}
	select {
	case <-p.stopCh:
	default:
		close(p.stopCh)
	}
	return nil
}

// Status returns the current proxy status.
func (p *proxy) Status() ProxyStatus {
	status := ProxyStatus{
		State: p.state,
		QueueSize: func() int {
			if p.queue == nil {
				return 0
			}
			return p.queue.Size()
		}(),
		LastRestartReason: p.lastRestartReason,
		LastError:         p.lastError,
	}

	if p.supervisor != nil {
		status.PID = p.supervisor.PID()
		status.Uptime = p.supervisor.Uptime()
		if p.maxRestartsInWnd != nil {
			status.RestartsInWindow = p.maxRestartsInWnd()
		}
		if p.maxRestarts != nil && status.RestartsInWindow > p.maxRestarts() {
			status.CanRecover = true
			if status.LastError == "" {
				status.LastError = "crash loop detected"
			}
		}
		if err := p.supervisor.LastError(); err != nil {
			status.LastError = err.Error()
		}
		if p.state == StateFailed {
			status.CanRecover = true
		}
	}

	return status
}

// loop is the main message processing loop.
func (p *proxy) loop() error {
	clientCh := make(chan []byte)
	childCh := make(chan []byte)
	errCh := make(chan error, 2)

	go p.readLoop(p.client, clientCh, errCh, "client")
	go p.readLoop(p.child, childCh, errCh, "child")

	for {
		select {
		case payload := <-clientCh:
			p.routeClientMessage(payload)
		case payload := <-childCh:
			p.handleChildMessage(payload)
		case err := <-errCh:
			if errors.Is(err, io.EOF) {
				return nil
			}
			p.crashLoop(err)
			if p.recorder != nil && p.crashExportEnabled {
				_ = p.recorder.Export(p.crashExportPath)
			}
			return err
		case <-p.stopCh:
			return nil
		}
	}
}

// readLoop continuously reads from a transport and forwards messages.
func (p *proxy) readLoop(
	t transport.Transport,
	out chan<- []byte,
	errCh chan<- error,
	label string,
) {
	defer func() {
		if r := recover(); r != nil && p.logger != nil {
			p.logger.Error("read panic", fmt.Errorf("%v", r))
		}
	}()

	for {
		payload, err := t.Read()
		if err != nil {
			if errors.Is(err, io.EOF) && label == "child" {
				errCh <- fmt.Errorf("child eof")
				return
			}
			errCh <- fmt.Errorf("%s read error: %w", label, err)
			return
		}

		if p.sanitizer != nil {
			if p.sanitizer.Classify(payload) != sanitizer.ChunkJSONRPC {
				if p.logger != nil {
					p.logger.Warn("discarding non-JSONRPC chunk", map[string]interface{}{
						"src": label,
					})
				}
				continue
			}
			payload = p.sanitizer.ValidateUTF8(payload)
		}

		out <- payload
	}
}
