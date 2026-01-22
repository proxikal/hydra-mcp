package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/proxikal/hydra/internal/injectable"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/security"
	"github.com/proxikal/hydra/internal/statestore"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/proxikal/hydra/internal/transport"
)

// Proxy orchestrates routing between client and child MCP server.
type Proxy interface {
	Run() error
	Shutdown() error
	Status() ProxyStatus
}

// ProxyStatus exposes current proxy metrics.
type ProxyStatus struct {
	State             supervisor.ServerState
	PID               int
	Uptime            time.Duration
	RestartsInWindow  int
	QueueSize         int
	CanRecover        bool
	LastRestartReason string
	LastError         string
}

// ProxyState mirrors supervisor.ServerState for readability.
type ProxyState = supervisor.ServerState

const (
	StateStopped    ProxyState = supervisor.StateStopped
	StateStarting   ProxyState = supervisor.StateStarting
	StateRunning    ProxyState = supervisor.StateRunning
	StateRestarting ProxyState = supervisor.StateRestarting
	StateFailed     ProxyState = supervisor.StateFailed
)

// Options configure proxy behavior.
type Options struct {
	QueueSize          int
	QueueTTL           time.Duration
	CollisionPolicy    string
	HydraTools         []injectable.ToolDefinition
	CrashExportPath    string
	CrashExportEnabled bool
}

// Dependencies for constructing a proxy.
type Dependencies struct {
	Logger              logger.Logger
	Sanitizer           sanitizer.Sanitizer
	Supervisor          supervisor.Supervisor
	Child               transport.Transport
	Client              transport.Transport
	Recorder            recorder.Recorder
	Redactor            security.Redactor
	StateStore          statestore.StateStore
	HydraTools          []injectable.ToolDefinition
	MaxRestartsInWindow func() int
	MaxRestarts         func() int
}

type proxy struct {
	state              ProxyState
	queue              Queue
	logger             logger.Logger
	sanitizer          sanitizer.Sanitizer
	supervisor         supervisor.Supervisor
	child              transport.Transport
	client             transport.Transport
	recorder           recorder.Recorder
	redactor           security.Redactor
	stateStore         statestore.StateStore
	hydraTools         []injectable.ToolDefinition
	collisionPolicy    string
	crashExportPath    string
	crashExportEnabled bool
	maxRestartsInWnd   func() int
	maxRestarts        func() int
	lastRestartReason  string
	lastError          string
	started            time.Time
	replay             func()
	stopCh             chan struct{}
	mainLoop           func() error
}

// New creates a new proxy instance.
func New(dep Dependencies, opts Options) Proxy {
	if opts.QueueSize <= 0 {
		opts.QueueSize = 100
	}
	if opts.QueueTTL <= 0 {
		opts.QueueTTL = 30 * time.Second
	}

	return &proxy{
		state:      StateStopped,
		queue:      NewQueue(opts.QueueSize, opts.QueueTTL),
		logger:     dep.Logger,
		sanitizer:  dep.Sanitizer,
		supervisor: dep.Supervisor,
		child:      dep.Child,
		client:     dep.Client,
		recorder:   dep.Recorder,
		redactor:   dep.Redactor,
		stateStore: dep.StateStore,
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

func (p *proxy) handleClientMessage(payload []byte) {
	p.record("client", payload, true)
	id := extractID(payload)

	switch p.state {
	case StateRunning:
		p.forwardToChild(payload)
	case StateStarting, StateRestarting:
		if err := p.queue.Enqueue(id, payload); err != nil {
			p.sendErrorResponse(id, "Server restarting, queue full")
			p.logger.Warn("restart queue full", map[string]interface{}{"id": id})
		}
	default:
		p.sendErrorResponse(id, "Server not running")
	}
}

func (p *proxy) drainQueue() {
	if p.queue == nil {
		return
	}
	for _, item := range p.queue.Drain() {
		p.forwardToChild(item.Payload)
	}
}

func (p *proxy) forwardToChild(payload []byte) {
	if p.child == nil {
		return
	}
	if err := p.child.Write(payload); err != nil && p.logger != nil {
		p.logger.Error("failed to forward to child", err)
		p.lastError = err.Error()
	}
}

func (p *proxy) sendErrorResponse(id interface{}, message string) {
	if p.client == nil {
		return
	}

	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32000,
			"message": message,
		},
	}
	if id != nil {
		resp["id"] = id
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = p.client.Write(data)
}

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

func (p *proxy) handleChildMessage(payload []byte) {
	p.record("child", payload, false)

	var msg recorder.JSONRPCMessage
	if err := json.Unmarshal(payload, &msg); err == nil {
		if p.state == StateStarting || p.state == StateRestarting {
			// Treat first result as initialize completion.
			if msg.Result != nil {
				p.handleInitializeResponse()
			}
		}

		if merged, ok := p.tryMergeTools(msg); ok {
			payload = merged
		}

		p.updateStateStore(msg)
	}

	p.forwardToClient(payload)
}

func (p *proxy) updateStateStore(msg recorder.JSONRPCMessage) {
	if p.stateStore == nil {
		return
	}

	if msg.Method == "initialize" && msg.Params != nil {
		p.stateStore.SetInitialize(msg.Params)
		return
	}

	if msg.Method == "resources/subscribe" && msg.Params != nil {
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(msg.Params, &params); err == nil && params.URI != "" {
			p.stateStore.AddSubscription(params.URI)
		}
		return
	}

	if msg.Method == "resources/unsubscribe" && msg.Params != nil {
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(msg.Params, &params); err == nil && params.URI != "" {
			p.stateStore.RemoveSubscription(params.URI)
		}
		return
	}
}

func (p *proxy) tryMergeTools(msg recorder.JSONRPCMessage) ([]byte, bool) {
	if msg.Result == nil || len(p.hydraTools) == 0 {
		return nil, false
	}

	var result map[string]interface{}
	if err := json.Unmarshal(msg.Result, &result); err != nil {
		return nil, false
	}

	rawTools, ok := result["tools"]
	if !ok {
		return nil, false
	}

	toolList, ok := rawTools.([]interface{})
	if !ok {
		return nil, false
	}

	childNames := make([]string, 0, len(toolList))
	for _, t := range toolList {
		switch v := t.(type) {
		case string:
			childNames = append(childNames, v)
		case map[string]interface{}:
			if name, ok := v["name"].(string); ok {
				childNames = append(childNames, name)
			}
		}
	}

	merged, err := mergeTools(childNames, p.hydraTools, p.collisionPolicy)
	if err != nil {
		if p.logger != nil {
			p.logger.Error("tool merge failed", err)
		}
		p.lastError = err.Error()
		p.sendErrorResponse(msg.ID, err.Error())
		if errors.Is(err, ErrCollision) {
			p.state = StateFailed
		}
		return nil, false
	}

	result["tools"] = merged
	p.notifyToolsChanged(merged)
	msg.Result, _ = json.Marshal(result)

	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, false
	}

	return payload, true
}

func (p *proxy) notifyToolsChanged(tools []injectable.ToolDefinition) {
	if p.client == nil {
		return
	}

	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}

	notification := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "tools/list_changed",
		"params": map[string]interface{}{
			"tools": names,
		},
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		return
	}
	_ = p.client.Write(payload)
}

func (p *proxy) forwardToClient(payload []byte) {
	if p.client == nil {
		return
	}

	if err := p.client.Write(payload); err != nil && p.logger != nil {
		p.logger.Error("failed to forward to client", err)
	}
}

func (p *proxy) record(direction string, payload []byte, isRequest bool) {
	if p.recorder == nil {
		return
	}

	var msg recorder.JSONRPCMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	if isRequest {
		p.recorder.RecordRequest(direction, msg)
		return
	}
	p.recorder.RecordResponse(direction, msg)
}

func extractID(payload []byte) interface{} {
	var obj map[string]interface{}
	if err := json.Unmarshal(payload, &obj); err != nil {
		return nil
	}
	if id, ok := obj["id"]; ok {
		return id
	}
	return nil
}
