# Hydra Architecture

This document contains detailed architecture specifications, algorithms, and implementation details.

---

## Component Diagram

### Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         AI Client                            │
│                    (Claude Desktop)                          │
└───────────────────────────┬─────────────────────────────────┘
                            │ stdio
                            ↓
┌───────────────────────────────────────────────────────────────┐
│                          HYDRA                                 │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │                      Transport                           │ │
│  │  - Protocol Detection (NDJSON/LSP)                      │ │
│  │  - UTF-8 Validation                                     │ │
│  └─────────────┬───────────────────────────────────────────┘ │
│                │                                               │
│  ┌─────────────▼───────────────────────────────────────────┐ │
│  │                      Sanitizer                           │ │
│  │  - JSON-RPC vs Pollution Classification                │ │
│  │  - Log Extraction                                       │ │
│  └─────────────┬───────────────────────────────────────────┘ │
│                │                                               │
│  ┌─────────────▼───────────────────────────────────────────┐ │
│  │                       Proxy                              │ │
│  │  - State Machine (STOPPED/STARTING/RUNNING/etc)        │ │
│  │  - Message Router                                       │ │
│  │  - Request Queue (RESTARTING state)                    │ │
│  │  - Tool Injection (hydra_*)                            │ │
│  └──┬────────────┬────────────┬──────────────┬────────────┘ │
│     │            │            │              │                │
│  ┌──▼──┐   ┌────▼────┐  ┌────▼─────┐  ┌────▼─────┐         │
│  │State│   │Watcher  │  │Recorder  │  │Injectable│         │
│  │Store│   │         │  │          │  │Tools     │         │
│  └─────┘   └────┬────┘  └──────────┘  └──────────┘         │
│                 │                                             │
│  ┌──────────────▼─────────────────────────────────────────┐ │
│  │                    Supervisor                            │ │
│  │  - Process Lifecycle (Start/Stop/Restart)              │ │
│  │  - Crash Loop Detection                                │ │
│  │  - Tree Kill                                           │ │
│  └───────────────────┬──────────────────────────────────────┘ │
└────────────────────────┼────────────────────────────────────────┘
                         │ stdio
                         ↓
              ┌──────────────────────┐
              │   Child MCP Server    │
              │  (Python/Node/Go)    │
              └──────────────────────┘
```

---

## Detailed State Machine

### State Diagram with Timing

```
STOPPED
  │
  │ hydra run
  ↓
STARTING (0-10s timeout)
  │
  ├─ initialize response received → RUNNING
  │
  └─ timeout (10s) → FAILED

RUNNING
  │
  ├─ File change detected → RESTARTING
  ├─ Process crash → RESTARTING
  ├─ Manual restart → RESTARTING
  └─ Clean shutdown → STOPPED

RESTARTING (target: 500ms, timeout: 10s)
  │
  ├─ initialize response + restarts ≤ max → RUNNING
  │
  └─ restarts > max in window → FAILED

FAILED
  │
  └─ hydra recover / hydra_force_restart → STOPPED
```

### State Transition Actions (Detailed)

#### STOPPED → STARTING

```
1. Validate config (command exists, cwd exists, etc)
2. Load .env file if specified
3. Merge environment variables
4. Spawn child process with SysProcAttr.Setpgid
5. Start watching child stdout/stderr
6. Start timeout timer (10s)
7. Transition to STARTING
8. Queue any client requests that arrive
```

#### STARTING → RUNNING

```
Trigger: Receive initialize response from child

1. Stop timeout timer
2. Cache initialize params in StateStore
3. Replay any queued requests to child
4. Send notifications/tools/list_changed to client
5. Transition to RUNNING
6. Start file watcher if enabled
```

#### RUNNING → RESTARTING

```
Trigger: File change / crash / manual restart

1. Pause file watcher
2. Record restart in tracker (timestamp)
3. Check if in crash loop (restarts > max in window)
   - If yes: goto FAILED
4. Execute pre_restart_command if enabled (with timeout)
   - If fails and on_error="abort": cancel restart, stay RUNNING
5. Send SIGTERM to child
6. Wait graceful_shutdown_ms (default: 2s)
7. If child still alive: Send SIGKILL
8. Wait for process exit
9. Spawn new child process
10. Transition to RESTARTING
11. Start timeout timer (10s)
```

#### RESTARTING → RUNNING

```
Trigger: Receive initialize response from child

1. Stop timeout timer
2. Replay initialize params from StateStore
3. Resurrect subscriptions:
   - For each cached subscription URI:
     - Send resources/subscribe to child
     - Update cache with new subscription ID
4. Drain request queue (FIFO order)
5. Send notifications/tools/list_changed to client
6. Resume file watcher
7. Reset cooldown timer
8. Transition to RUNNING
```

#### RESTARTING → FAILED

```
Trigger: Restart counter > max_restarts in window OR timeout

1. Kill child process (SIGKILL)
2. Send notification to client:
   {
     "method": "notifications/message",
     "params": {
       "level": "error",
       "message": "Server in crash loop (X restarts in Ys). Use hydra_status."
     }
   }
3. Flush request queue (send error responses)
4. Transition to FAILED
5. Expose hydra_force_restart tool
```

#### FAILED → STOPPED

```
Trigger: hydra recover CLI OR hydra_force_restart tool

1. Reset restart counter
2. Clear StateStore
3. Transition to STOPPED
4. Ready for hydra run
```

---

## Request Lifecycle Diagrams

### Normal Operation (RUNNING)

```
Time    Client              Hydra                   Child
─────   ──────              ─────                   ─────
T=0     initialize ──────►  validate, forward ───►  process
T=50                        ◄─── response ──────────  return
T=50    ◄─── response ─────  forward

T=100   tools/list ──────►  forward ──────────────►  process
T=150                       ◄─── child tools ────────  return
T=150                        merge hydra_* tools
T=150   ◄─── merged ───────  return

T=200   tools/call ───────►  forward ──────────────►  execute
T=500                       ◄─── result ─────────────  return
T=500                        check size (< 50KB)
T=500   ◄─── result ───────  forward
```

### Restart Operation (RESTARTING)

```
Time    Client              Hydra                   Child (Old → New)
─────   ──────              ─────                   ─────
T=0                          [File change detected]
T=0                          Pause watcher
T=0                          RUNNING → RESTARTING
T=0                          Start debounce (500ms)

T=100                        [Another file change]
T=100                        Reset debounce → T=600ms

T=200   tools/call(id=5) ►  Queue (size: 1)

T=300   tools/call(id=6) ►  Queue (size: 2)

T=600                        Debounce expires
T=600                        Pre-restart hook runs ──►  [build script]
T=1200                       ◄─── hook done ───────────  [exit 0]
T=1200                       SIGTERM ──────────────────►  [old child]
T=3200                       [graceful_shutdown_ms]
T=3200                       SIGKILL ──────────────────►  [old child dies]
T=3300                       Spawn new child ──────────►  [new child starts]

T=3800                       ◄─── initialize ───────────  [new child ready]
T=3800                       Replay init params ────────►
T=3900                       ◄─── init response ─────────
T=3900                       Resurrect subscriptions ──►
T=4000                       ◄─── subscription IDs ──────
T=4000                       Drain queue:
T=4000                       Forward id=5 ─────────────►  execute
T=4100                       ◄─── result(id=5) ──────────
T=4100   ◄─── result(id=5)  Forward
T=4100                       Forward id=6 ─────────────►  execute
T=4200                       ◄─── result(id=6) ──────────
T=4200   ◄─── result(id=6)  Forward
T=4200                       Send tools/list_changed ►
T=4200                       RESTARTING → RUNNING
T=4200                       Resume file watcher
```

### Queue Overflow (RESTARTING)

```
Time    Client              Hydra (Queue: 0/100)
─────   ──────              ─────
T=0                          RESTARTING
T=0     req(id=1) ────────►  Queue (1/100)
T=10    req(id=2) ────────►  Queue (2/100)
...
T=990   req(id=100) ──────►  Queue (100/100) [FULL]

T=1000  req(id=101) ──────►  QUEUE FULL
T=1000  ◄─── error ────────  { "code": -32000,
                                "message": "Server restarting, queue full" }

T=2000                       [Child ready]
T=2000                       Drain queue (id=1..100)
T=2000                       RUNNING
```

---

## Algorithm Specifications

### File Watching: Debounce + Batching

**Goal:** Batch rapid file changes (git checkout, npm install) into single restart.

**Algorithm:**

```go
type Debouncer struct {
    debounceMs         int
    batchWindowMs      int
    cooldownMs         int

    debounceTimer      *time.Timer
    batchStartTime     *time.Time
    lastRestartTime    time.Time
}

func (d *Debouncer) OnFileChange(path string) {
    now := time.Now()

    // Cooldown check: Don't restart if we just restarted
    if now.Sub(d.lastRestartTime) < time.Duration(d.cooldownMs)*time.Millisecond {
        log.Debug("Ignoring file change during cooldown", "path", path)
        return
    }

    // Batch window check: Force restart if window expired
    if d.batchStartTime == nil {
        now := time.Now()
        d.batchStartTime = &now
    }

    if now.Sub(*d.batchStartTime) > time.Duration(d.batchWindowMs)*time.Millisecond {
        log.Debug("Batch window expired, forcing restart")
        d.TriggerRestart()
        return
    }

    // Reset debounce timer
    if d.debounceTimer != nil {
        d.debounceTimer.Stop()
    }

    d.debounceTimer = time.AfterFunc(
        time.Duration(d.debounceMs)*time.Millisecond,
        d.TriggerRestart,
    )
}

func (d *Debouncer) TriggerRestart() {
    d.batchStartTime = nil
    d.debounceTimer = nil

    // ... actual restart logic ...

    d.lastRestartTime = time.Now()
}
```

**Example Timeline:**

```
T=0ms     main.py changed → batchStart=0, debounce=500ms
T=100ms   utils.py changed → debounce=600ms (reset)
T=300ms   tests.py changed → debounce=800ms (reset)
T=800ms   Debounce expires → RESTART
T=1500ms  Restart completes → lastRestart=1500ms
T=1600ms  main.py changed → IGNORED (cooldown: 1600 < 1500+5000)
T=6500ms  main.py changed → ALLOWED (6500 > 1500+5000)
```

### Protocol Detection

**Goal:** Auto-detect NDJSON vs LSP-style framing.

**Algorithm:**

```go
func DetectProtocol(reader io.Reader, timeout time.Duration) (Protocol, error) {
    // Buffer first chunk
    buf := make([]byte, 1024)

    // Read with timeout
    readChan := make(chan int)
    go func() {
        n, _ := reader.Read(buf)
        readChan <- n
    }()

    var n int
    select {
    case n = <-readChan:
        // Got data
    case <-time.After(timeout):
        // Timeout, use config default or fallback
        return ProtocolNDJSON, nil
    }

    chunk := buf[:n]

    // Check for LSP header
    if bytes.HasPrefix(chunk, []byte("Content-Length:")) {
        return ProtocolLSP, nil
    }

    // Check for NDJSON
    trimmed := bytes.TrimSpace(chunk)
    if bytes.HasPrefix(trimmed, []byte("{")) {
        return ProtocolNDJSON, nil
    }

    // Unknown
    return ProtocolUnknown, fmt.Errorf("unable to detect protocol")
}
```

**Fallback Strategy:**

```
1. Try auto-detect with 1s timeout
2. If detected → use detected protocol
3. If timeout → use config override OR default to NDJSON
4. If unknown → try NDJSON first, then LSP
5. If both fail → error to client, enter FAILED state
```

### Sanitizer: STDIO Pollution Filter

**Goal:** Separate valid JSON-RPC from debug logs/stack traces.

**Classification:**

```go
type ChunkType int

const (
    ChunkEmpty ChunkType = iota
    ChunkJSONRPC
    ChunkPollution
)

func ClassifyChunk(chunk []byte) ChunkType {
    trimmed := bytes.TrimSpace(chunk)

    // Empty
    if len(trimmed) == 0 {
        return ChunkEmpty
    }

    // Try parse as JSON
    var msg map[string]interface{}
    if err := json.Unmarshal(trimmed, &msg); err != nil {
        // Not JSON → pollution
        return ChunkPollution
    }

    // Check for JSON-RPC 2.0 marker
    if msg["jsonrpc"] == "2.0" {
        return ChunkJSONRPC
    }

    // Valid JSON but not JSON-RPC → pollution
    return ChunkPollution
}
```

**Handling:**

```go
func HandleChunk(chunk []byte, chunkType ChunkType) {
    switch chunkType {
    case ChunkEmpty:
        // Ignore

    case ChunkJSONRPC:
        // Forward to client
        client.Write(chunk)

    case ChunkPollution:
        // Option 1: Convert to MCP log notification
        notification := JSONRPCNotification{
            JSONRPC: "2.0",
            Method:  "notifications/message",
            Params: map[string]interface{}{
                "level":  "info",
                "logger": "child-stdout",
                "data":   TruncateLog(string(chunk), 1000),
            },
        }
        client.Write(json.Marshal(notification))

        // Option 2: Discard
        log.Warn("Discarded stdout pollution", "content", string(chunk))
    }
}
```

### Crash Loop Detection

**Goal:** Detect when server is in restart death spiral.

**Algorithm:**

```go
type RestartTracker struct {
    mu              sync.Mutex
    restarts        []time.Time
    windowSeconds   int
    maxRestarts     int
}

func (rt *RestartTracker) RecordRestart() {
    rt.mu.Lock()
    defer rt.mu.Unlock()

    now := time.Now()
    rt.restarts = append(rt.restarts, now)

    // Remove restarts outside window
    cutoff := now.Add(-time.Duration(rt.windowSeconds) * time.Second)

    filtered := []time.Time{}
    for _, t := range rt.restarts {
        if t.After(cutoff) {
            filtered = append(filtered, t)
        }
    }
    rt.restarts = filtered
}

func (rt *RestartTracker) InCrashLoop() bool {
    rt.mu.Lock()
    defer rt.mu.Unlock()

    return len(rt.restarts) > rt.maxRestarts
}

func (rt *RestartTracker) Reset() {
    rt.mu.Lock()
    defer rt.mu.Unlock()

    rt.restarts = []time.Time{}
}
```

**Example:**

```
Config: max_restarts=10, window=60s

T=0s    Restart #1  → restarts=[0]
T=5s    Restart #2  → restarts=[0,5]
T=10s   Restart #3  → restarts=[0,5,10]
...
T=50s   Restart #11 → restarts=[0,5,10,15,...,50]
T=50s   len=11 > 10 → IN CRASH LOOP → FAILED state

T=100s  User fixes code, runs: hydra recover
T=100s  Reset() → restarts=[]
T=100s  FAILED → STOPPED
```

### Subscription Resurrection

**Goal:** Re-subscribe to resources after restart (IDs change).

**State:**

```go
type SubscriptionCache struct {
    mu    sync.Mutex
    subs  map[string]string  // URI → current subscription ID
}

// On client resources/subscribe:
func (sc *SubscriptionCache) Add(uri string) {
    sc.mu.Lock()
    defer sc.mu.Unlock()

    sc.subs[uri] = ""  // Don't know ID yet
}

// After restart, before draining queue:
func (sc *SubscriptionCache) Resurrect(child ChildProcess) error {
    sc.mu.Lock()
    uris := make([]string, 0, len(sc.subs))
    for uri := range sc.subs {
        uris = append(uris, uri)
    }
    sc.mu.Unlock()

    for _, uri := range uris {
        // Send resources/subscribe to child
        resp, err := child.Subscribe(uri)
        if err != nil {
            // Resource no longer exists, remove from cache
            sc.Remove(uri)
            continue
        }

        // Update cache with new ID
        sc.mu.Lock()
        sc.subs[uri] = resp.SubscriptionID
        sc.mu.Unlock()
    }

    return nil
}
```

---

## Interface Specifications

### Transport

```go
type Transport interface {
    Read() ([]byte, error)
    Write([]byte) error
    Close() error
}

type Protocol int

const (
    ProtocolNDJSON Protocol = iota
    ProtocolLSP
)
```

### Sanitizer

```go
type Sanitizer interface {
    Classify(chunk []byte) ChunkType
    ExtractJSONRPC(chunk []byte) ([]byte, error)
}
```

### Supervisor

```go
type ServerState int

const (
    StateStopped ServerState = iota
    StateStarting
    StateRunning
    StateRestarting
    StateFailed
)

type Supervisor interface {
    Start() error
    Stop() error
    Restart() error
    State() ServerState
    PID() int
    Uptime() time.Duration
    LastError() error
}
```

### StateStore

```go
type StateStore interface {
    SetInitialize(params json.RawMessage)
    GetInitialize() json.RawMessage

    AddSubscription(uri string)
    RemoveSubscription(uri string)
    GetSubscriptions() []string
    UpdateSubscriptionID(uri string, id string)

    Clear()
}
```

### Watcher

```go
type WatchEvent struct {
    Path      string
    Timestamp time.Time
}

type Watcher interface {
    Start() error
    Stop() error
    Events() <-chan WatchEvent
}
```

### Recorder

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

### Proxy

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

---

## Error Handling Strategy

### Error Wrapping

All errors MUST be wrapped with context:

```go
if err != nil {
    return fmt.Errorf("failed to start child process: %w", err)
}
```

### Error Propagation

```
Component Error → Log to stderr + Transition to FAILED state + Send error to client
```

**Never:**
- Silent failures
- Panic (except in defer recover() blocks)
- Exit without logging

**Always:**
- Wrap errors with context
- Log to stderr (never stdout)
- Send JSON-RPC error to client

### Panic Recovery

Main proxy loop MUST have panic recovery:

```go
func (p *proxy) Run() error {
    defer func() {
        if r := recover(); r != nil {
            p.logger.Error("Proxy panic", "panic", r, "stack", string(debug.Stack()))

            // Send generic error to client
            p.client.Write(JSONRPCError{
                Code:    -32603,
                Message: "Internal server error",
            })

            // Do NOT exit Hydra
        }
    }()

    // Main loop...
}
```

---

**End of Architecture Documentation**
