# Phase 3: Orchestration (Week 3)

**Goal:** Implement proxy (orchestrator), injectable tools, and traffic recorder.

---

## Tasks

### 1. Proxy Package

**Files:**
- `internal/proxy/proxy.go` - Main proxy loop
- `internal/proxy/router.go` - Message routing
- `internal/proxy/queue.go` - Request queue (RESTARTING state)
- `internal/proxy/tools.go` - Tool injection/merging
- `internal/proxy/statemachine.go` - State transitions
- `internal/proxy/proxy_test.go` - Tests

**Interface:**
```go
type Proxy interface {
    Run() error
    Shutdown() error
    Status() ProxyStatus
}

type ProxyStatus struct {
    State              ServerState
    PID                int
    Uptime             time.Duration
    RestartsInWindow   int
    QueueSize          int
    LastRestartReason  string
    LastError          string
    CanRecover         bool
}
```

**Implementation:**
- Main event loop (select on channels)
- State machine (STOPPED → STARTING → RUNNING → RESTARTING → FAILED)
- Request routing (client ↔ child)
- Request queueing during RESTARTING
- Tool list merging (child tools + hydra_* tools)
- Panic recovery in main loop

**State Transitions:**
```go
func (p *proxy) handleFileChange() {
    if p.state != StateRunning {
        return
    }

    p.state = StateRestarting
    p.supervisor.Restart()
}

func (p *proxy) handleInitializeResponse() {
    if p.state == StateStarting || p.state == StateRestarting {
        p.replayState()
        p.drainQueue()
        p.state = StateRunning
    }
}
```

**Tests:**
- Normal request/response flow (RUNNING)
- Request queued during STARTING
- Request queued during RESTARTING
- Queue drained after restart
- Queue overflow → error response
- Tool list merging
- Namespace collision detection
- State transitions
- Panic recovery

---

### 2. Injectable Package

**Files:**
- `internal/injectable/tools.go` - Tool definitions
- `internal/injectable/restart.go` - hydra_restart impl
- `internal/injectable/status.go` - hydra_status impl
- `internal/injectable/logs.go` - hydra_logs impl
- `internal/injectable/force.go` - hydra_force_restart impl
- `internal/injectable/injectable_test.go` - Tests

**Interface:**
```go
type InjectableTools interface {
    GetDefinitions() []ToolDefinition
    Handle(toolName string, params map[string]interface{}) (interface{}, error)
}

type ToolDefinition struct {
    Name        string
    Description string
    InputSchema map[string]interface{}
}
```

**Implementation:**
- Tool definitions (JSON schema)
- Tool handlers
- Collision detection
- Namespace enforcement (hydra_* prefix)

**Tests:**
- hydra_restart triggers restart
- hydra_status returns correct state
- hydra_logs returns recent logs
- hydra_logs redacts secrets
- hydra_force_restart resets counter
- Namespace collision detected

---

### 3. Recorder Package

**Files:**
- `internal/recorder/recorder.go` - Interface
- `internal/recorder/buffer.go` - Circular buffer
- `internal/recorder/export.go` - Export to JSON
- `internal/recorder/recorder_test.go` - Tests

**Interface:**
```go
type JSONRPCMessage struct {
    JSONRPC string          `json:"jsonrpc"`
    Method  string          `json:"method,omitempty"`
    ID      interface{}     `json:"id,omitempty"`
    Params  json.RawMessage `json:"params,omitempty"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *JSONRPCError   `json:"error,omitempty"`
}

type Recorder interface {
    RecordRequest(direction string, msg JSONRPCMessage)
    RecordResponse(direction string, msg JSONRPCMessage)
    Export(path string) error
}
```

**Implementation:**
- Circular buffer (fixed size: 50)
- Privacy-first (disabled by default)
- Redaction before export
- Body inclusion flags

**Tests:**
- Record request
- Record response
- Buffer overflow (keeps last N)
- Export to file
- Redaction in export
- Body inclusion/exclusion flags

---

### 4. Integration Tests

**Files:**
- `test/fixtures/echo_server.py` - Echo MCP server
- `test/fixtures/crash_server.py` - Crash-on-startup server
- `test/integration/restart_test.go` - Restart scenarios
- `test/integration/crash_loop_test.go` - Crash loop detection
- `test/integration/queue_test.go` - Request queueing

**Fixtures:**

**echo_server.py:**
```python
#!/usr/bin/env python3
import json
import sys

def main():
    for line in sys.stdin:
        req = json.loads(line)
        if req.get("method") == "initialize":
            resp = {
                "jsonrpc": "2.0",
                "id": req["id"],
                "result": {"protocolVersion": "1.0", "serverInfo": {"name": "echo"}}
            }
        else:
            resp = {
                "jsonrpc": "2.0",
                "id": req.get("id"),
                "result": req.get("params", {})
            }
        print(json.dumps(resp), flush=True)

if __name__ == "__main__":
    main()
```

**crash_server.py:**
```python
#!/usr/bin/env python3
import sys
sys.exit(1)
```

**Integration Tests:**
- Restart preserves state (initialize params)
- Crash loop detection works
- Request queueing during restart
- Subscription resurrection

---

## Definition of Done (Phase 3)

- [ ] Proxy package: 95%+ coverage (critical path)
- [ ] Injectable package: 90%+ coverage
- [ ] Recorder package: 90%+ coverage
- [ ] Test fixtures created and working
- [ ] Integration tests pass
- [ ] All tests pass with `-race`
- [ ] golangci-lint passes
- [ ] Can run full lifecycle: start → restart → crash loop → recover

---

## Files Created (Phase 3)

```
internal/proxy/proxy.go
internal/proxy/router.go
internal/proxy/queue.go
internal/proxy/tools.go
internal/proxy/statemachine.go
internal/proxy/proxy_test.go
internal/injectable/tools.go
internal/injectable/restart.go
internal/injectable/status.go
internal/injectable/logs.go
internal/injectable/force.go
internal/injectable/injectable_test.go
internal/recorder/recorder.go
internal/recorder/buffer.go
internal/recorder/export.go
internal/recorder/recorder_test.go
test/fixtures/echo_server.py
test/fixtures/crash_server.py
test/integration/restart_test.go
test/integration/crash_loop_test.go
test/integration/queue_test.go
```

---

## Estimated Time

**5-7 days**
