# Phase 2: Core Logic (Week 2)

**Goal:** Implement supervisor, state management, file watching, and security features.

---

## Tasks

### 1. Supervisor Package

**Files:**
- `internal/supervisor/supervisor.go` - Interface
- `internal/supervisor/process.go` - exec.Cmd wrapper
- `internal/supervisor/signals.go` - SIGTERM/SIGKILL handling
- `internal/supervisor/treekill.go` - Cross-platform tree kill
- `internal/supervisor/restart.go` - Restart counter, crash loop
- `internal/supervisor/supervisor_test.go` - Tests

**Interface:**
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
    ResetRestartCounter()
}
```

**Implementation:**
- Spawn child with `SysProcAttr.Setpgid` (Unix)
- Tree kill using gopsutil
- Graceful shutdown: SIGTERM → wait → SIGKILL
- Restart counter with time window
- Crash loop detection

**Tests:**
- Start process successfully
- Start with invalid command → error
- Stop running process
- Restart increments counter
- Crash loop detection (> max_restarts)
- Crash loop window expiration
- Tree kill works (spawn `sleep 1000` from bash)

---

### 2. StateStore Package

**Files:**
- `internal/statestore/statestore.go` - Interface
- `internal/statestore/initialize.go` - Initialize params cache
- `internal/statestore/subscriptions.go` - Subscription resurrection
- `internal/statestore/statestore_test.go` - Tests

**Interface:**
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

**Implementation:**
- Thread-safe (sync.Mutex)
- Initialize params stored as json.RawMessage (no unmarshalling)
- Subscriptions stored as map[uri]subscriptionID

**Tests:**
- SetInitialize → GetInitialize
- AddSubscription → GetSubscriptions
- RemoveSubscription
- UpdateSubscriptionID
- Clear all state
- Concurrent access (race detector)

---

### 3. Watcher Package

**Files:**
- `internal/watcher/watcher.go` - Interface
- `internal/watcher/debounce.go` - Debounce + batching logic
- `internal/watcher/gitignore.go` - Gitignore parser integration
- `internal/watcher/watcher_test.go` - Tests

**Interface:**
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

**Implementation:**
- Use fsnotify for file system events
- Debounce algorithm (see ARCHITECTURE.md)
- Gitignore integration (sabhiram/go-gitignore)
- Batch window and cooldown logic

**Tests:**
- Single file change triggers event
- Rapid changes batched into one event
- Batch window expiration forces restart
- Cooldown prevents immediate restart
- Gitignore patterns work
- Stop watcher cleanly

---

### 4. Security Package

**Files:**
- `internal/security/redact.go` - Secret redaction
- `internal/security/ratelimit.go` - Log rate limiter
- `internal/security/sizelimit.go` - Payload size limiter
- `internal/security/security_test.go` - Tests

**Interfaces:**
```go
type Redactor interface {
    Redact(content string, patterns []string) string
}

type RateLimiter interface {
    Allow() bool
}

type SizeLimiter interface {
    Limit(response JSONRPCResponse, maxKB int) JSONRPCResponse
}
```

**Implementation:**
- Regex-based secret redaction
- Token bucket rate limiter
- Payload size check with truncation

**Tests:**
- Redact OpenAI API key (sk-...)
- Redact generic API_KEY
- Redact password (case-insensitive)
- Rate limiter allows burst
- Rate limiter blocks after burst
- Rate limiter refills over time
- Size limiter passes small payloads
- Size limiter truncates large payloads

---

## Definition of Done (Phase 2)

- [ ] Supervisor package: 95%+ coverage (critical path)
- [ ] StateStore package: 90%+ coverage
- [ ] Watcher package: 90%+ coverage
- [ ] Security package: 90%+ coverage
- [ ] All tests pass with `-race`
- [ ] golangci-lint passes
- [ ] Mocks generated for all interfaces
- [ ] Integration test: Spawn real process and restart it

---

## Files Created (Phase 2)

```
internal/supervisor/supervisor.go
internal/supervisor/process.go
internal/supervisor/signals.go
internal/supervisor/treekill.go
internal/supervisor/restart.go
internal/supervisor/supervisor_test.go
internal/statestore/statestore.go
internal/statestore/initialize.go
internal/statestore/subscriptions.go
internal/statestore/statestore_test.go
internal/watcher/watcher.go
internal/watcher/debounce.go
internal/watcher/gitignore.go
internal/watcher/watcher_test.go
internal/security/redact.go
internal/security/ratelimit.go
internal/security/sizelimit.go
internal/security/security_test.go
```

---

## Estimated Time

**5-7 days**
