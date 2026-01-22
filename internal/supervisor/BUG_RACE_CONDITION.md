# Supervisor Race Condition Bug

## The Problem

Two goroutines trying to wait on the same process:
1. `stopProcess()` does `process.Wait()`
2. `monitor()` does `process.Wait()`

**Result:** Second waiter gets "wait: no child processes" error.

## When It Happens

```
Restart() called
  ↓
stopProcess() kills process + waits
  ↓
Old monitor() ALSO waits (RACE!)
  ↓
monitor() sets state = Stopped (overwrites Running!)
```

## The Fix

**Add context to cancel old monitor before restarting:**

```go
type supervisor struct {
    // ... existing fields ...
    monitorCancel context.CancelFunc  // NEW
}

func (s *supervisor) Start() error {
    // Create cancelable context for monitor
    ctx, cancel := context.WithCancel(context.Background())
    s.monitorCancel = cancel

    // Start monitor with context
    go s.monitor(ctx)
}

func (s *supervisor) stopProcess() error {
    // Cancel old monitor FIRST
    if s.monitorCancel != nil {
        s.monitorCancel()
    }

    // Then stop process (only ONE waiter now)
    // ... rest of existing code ...
}

func (s *supervisor) monitor(ctx context.Context) {
    // Exit if context cancelled
    select {
    case <-ctx.Done():
        return
    default:
    }

    // Wait for process
    err := s.process.Wait()

    // Check context again before changing state
    select {
    case <-ctx.Done():
        return  // Don't touch state if cancelled
    default:
    }

    // Safe to update state now
    s.mu.Lock()
    defer s.mu.Unlock()
    // ... rest of existing code ...
}
```

## Changes Required

1. Add `context` import
2. Add `monitorCancel context.CancelFunc` field to supervisor struct
3. Update `Start()` to create context and pass to monitor
4. Update `stopProcess()` to cancel context before stopping
5. Update `monitor()` to accept context and check cancellation

**Files:** `internal/supervisor/process.go`
