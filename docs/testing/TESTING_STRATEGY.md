# Hydra Testing Strategy

Complete TDD approach, testing layers, and requirements for Hydra.

---

## TDD Philosophy

**Red-Green-Refactor** cycle is mandatory for all components:

1. **Red** - Write failing test
2. **Green** - Write minimal code to pass
3. **Refactor** - Clean up while keeping tests green
4. **Repeat**

---

## Testing Layers

### Layer 1: Unit Tests

**Scope:** Individual components in isolation

**Coverage Target:** 80%+ (95%+ for critical paths)

**Tools:**
- `testing` (Go standard library)
- `github.com/stretchr/testify` (assertions)
- `github.com/vektra/mockery` (mock generation)

**Mocking Strategy:**
- All interfaces have generated mocks
- Use mocks for external dependencies (OS, network, time)
- Never mock internal package code

**Example:**

```go
// internal/sanitizer/sanitizer_test.go
func TestClassifyChunk(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected ChunkType
    }{
        {
            name:     "valid_jsonrpc",
            input:    `{"jsonrpc":"2.0","method":"initialize"}`,
            expected: ChunkJSONRPC,
        },
        {
            name:     "pollution",
            input:    `Debug: Starting server...`,
            expected: ChunkPollution,
        },
        {
            name:     "empty",
            input:    ``,
            expected: ChunkEmpty,
        },
    }

    s := NewSanitizer()

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := s.Classify([]byte(tt.input))
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

---

### Layer 2: Integration Tests

**Scope:** Multiple components working together

**Coverage Target:** 100% of component interactions

**Tools:**
- Test fixtures (echo_server.py, crash_server.py)
- Real child processes
- In-memory transports

**Example:**

```go
// test/integration/restart_test.go
func TestRestartPreservesState(t *testing.T) {
    // Start Hydra with echo_server.py
    hydra := startHydra(t, "test/fixtures/echo_server.py")
    defer hydra.Stop()

    // Send initialize
    initResp := hydra.SendRequest(`{"jsonrpc":"2.0","method":"initialize","id":1}`)
    assert.NotNil(t, initResp)

    // Trigger restart
    hydra.TriggerRestart()

    // Wait for RUNNING
    waitForState(t, hydra, StateRunning, 5*time.Second)

    // Send request - should work (state was replayed)
    resp := hydra.SendRequest(`{"jsonrpc":"2.0","method":"tools/list","id":2}`)
    assert.NotNil(t, resp)
}
```

---

### Layer 3: Chaos Tests

**Scope:** Real-world failure scenarios

**Coverage Target:** All known failure modes

**See:** `docs/testing/TEST_SCENARIOS.md` for complete list

**Example:**

```go
// test/integration/crash_loop_test.go
func TestCrashLoop(t *testing.T) {
    cfg := &config.Server{
        Command:     "python",
        Args:        []string{"test/fixtures/crash_server.py"},
        MaxRestarts: 3,
    }

    hydra := startHydra(t, cfg)
    defer hydra.Stop()

    // Wait for FAILED state
    waitForState(t, hydra, StateFailed, 10*time.Second)

    status := hydra.Status()
    assert.Equal(t, StateFailed, status.State)
    assert.Greater(t, status.RestartsInWindow, 3)
}
```

---

### Layer 4: Benchmark Tests

**Scope:** Performance validation

**Coverage Target:** All performance targets from PRD

**Example:**

```go
// test/benchmarks/proxy_test.go
func BenchmarkProxyOverhead(b *testing.B) {
    hydra := startHydra(b, "test/fixtures/echo_server.py")
    defer hydra.Stop()

    msg := []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        hydra.SendRequest(string(msg))
        latency := time.Since(start)

        // Record latency
        if latency > 50*time.Millisecond {
            b.Errorf("Latency %v > 50ms", latency)
        }
    }
}
```

---

## Test Organization

```
/test
  /fixtures          # Test MCP servers
    echo_server.py
    crash_server.py
    slow_server.py
    chatty_server.py

  /integration       # Integration tests
    restart_test.go
    crash_loop_test.go
    file_watch_test.go
    subscription_test.go
    queue_test.go

  /unit              # Cross-package unit tests
    sanitizer_test.go
    statestore_test.go
    supervisor_test.go
    debounce_test.go

  /benchmarks        # Performance tests
    proxy_test.go
    restart_test.go
```

**Package-Level Tests:**

Each `/internal/{package}` has its own `*_test.go` files:

```
/internal/sanitizer
  sanitizer.go
  sanitizer_test.go
  classifier.go
  classifier_test.go
```

---

## Test Fixtures

### echo_server.py

Minimal MCP server that echoes requests:

```python
#!/usr/bin/env python3
import json
import sys

def main():
    # Read initialize
    line = sys.stdin.readline()
    req = json.loads(line)

    # Send initialize response
    resp = {
        "jsonrpc": "2.0",
        "id": req["id"],
        "result": {
            "protocolVersion": "1.0",
            "serverInfo": {"name": "echo-server"}
        }
    }
    print(json.dumps(resp), flush=True)

    # Echo loop
    for line in sys.stdin:
        req = json.loads(line)
        resp = {
            "jsonrpc": "2.0",
            "id": req.get("id"),
            "result": req.get("params", {})
        }
        print(json.dumps(resp), flush=True)

if __name__ == "__main__":
    main()
```

### crash_server.py

Server that crashes on startup:

```python
#!/usr/bin/env python3
import sys

sys.exit(1)  # Always crash
```

### slow_server.py

Server with 30s startup time:

```python
#!/usr/bin/env python3
import json
import sys
import time

time.sleep(30)  # Simulate slow startup

# Then normal echo server
# ...
```

### chatty_server.py

Server that spams logs:

```python
#!/usr/bin/env python3
import json
import sys
import time

def main():
    # Send initialize response
    # ...

    # Spam logs
    while True:
        print("DEBUG: " + "X" * 1000, file=sys.stderr, flush=True)
        time.sleep(0.01)  # 100 logs/second

if __name__ == "__main__":
    main()
```

---

## Mock Strategy

### Generate Mocks with Mockery

```bash
# Generate mock for Supervisor interface
mockery --name=Supervisor --dir=internal/supervisor --output=internal/supervisor/mocks

# Generate all mocks
make mocks
```

**Makefile:**
```makefile
.PHONY: mocks
mocks:
	mockery --all --dir=internal --output=internal/mocks
```

### Using Mocks in Tests

```go
// internal/proxy/proxy_test.go
func TestProxyRestart(t *testing.T) {
    mockSupervisor := new(mocks.Supervisor)
    mockStateStore := new(mocks.StateStore)

    // Setup expectations
    mockSupervisor.On("Restart").Return(nil)
    mockSupervisor.On("State").Return(StateRunning)

    // Create proxy with mocks
    proxy := NewProxy(mockSupervisor, mockStateStore)

    // Test
    err := proxy.HandleRestart()
    assert.NoError(t, err)

    // Verify expectations
    mockSupervisor.AssertExpectations(t)
}
```

---

## Coverage Requirements

### Minimum Coverage

- **Overall:** 80%
- **Critical paths (proxy, supervisor):** 95%
- **Unit tests:** 90%
- **Integration tests:** 100% of component interactions

### CI Enforcement

```bash
# Run tests with coverage
go test -race -coverprofile=coverage.out ./...

# Check coverage threshold
go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | \
    awk '{if ($1 < 80) {print "Coverage below 80%"; exit 1}}'
```

**GitHub Actions:**
```yaml
- name: Test
  run: |
    go test -race -coverprofile=coverage.out ./...

- name: Check Coverage
  run: |
    coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$coverage < 80" | bc -l) )); then
      echo "Coverage $coverage% is below 80%"
      exit 1
    fi
```

---

## Test Execution

### Local Development

```bash
# Run all tests
make test

# Run with race detector
make test-race

# Run specific package
go test ./internal/proxy/...

# Run specific test
go test -run TestProxyRestart ./internal/proxy/...

# Run with verbose output
go test -v ./...
```

### CI Pipeline

```yaml
# .github/workflows/test.yml
name: Test
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Unit Tests
        run: go test -race -coverprofile=coverage.out ./...

      - name: Integration Tests
        run: go test -race ./test/integration/...

      - name: Chaos Tests
        run: go test -race ./test/integration/... -tags=chaos

      - name: Benchmarks
        run: go test -bench=. -benchmem ./test/benchmarks/...

      - name: Coverage Check
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 80" | bc -l) )); then
            exit 1
          fi
```

---

## Test Data Management

### Isolation

Each test should:
- Create temp directories
- Clean up after itself
- Not depend on global state

**Example:**

```go
func TestFileWatcher(t *testing.T) {
    // Create temp dir
    tmpDir, err := os.MkdirTemp("", "hydra-test-*")
    require.NoError(t, err)
    defer os.RemoveAll(tmpDir)

    // Create test file
    testFile := filepath.Join(tmpDir, "test.txt")
    os.WriteFile(testFile, []byte("test"), 0644)

    // Test watcher
    // ...
}
```

### Parallel Tests

Tests that don't share state should run in parallel:

```go
func TestSanitizer(t *testing.T) {
    t.Parallel()  // Run in parallel with other tests

    // Test...
}
```

**Don't parallelize:**
- Tests that modify global state
- Tests that spawn child processes
- Tests that use the same temp directory

---

## TDD Workflow Example

### Feature: Crash Loop Detection

#### Step 1: Write Failing Test (Red)

```go
func TestCrashLoop(t *testing.T) {
    tracker := NewRestartTracker(3, 60)  // max=3, window=60s

    tracker.RecordRestart()
    tracker.RecordRestart()
    tracker.RecordRestart()
    tracker.RecordRestart()  // 4th restart

    assert.True(t, tracker.InCrashLoop())  // FAILS - InCrashLoop not implemented
}
```

**Run:** `go test ./internal/supervisor/...`

**Result:** ❌ Test fails (method doesn't exist)

#### Step 2: Write Minimal Code (Green)

```go
type RestartTracker struct {
    restarts      []time.Time
    maxRestarts   int
    windowSeconds int
}

func (rt *RestartTracker) RecordRestart() {
    rt.restarts = append(rt.restarts, time.Now())
}

func (rt *RestartTracker) InCrashLoop() bool {
    return len(rt.restarts) > rt.maxRestarts
}
```

**Run:** `go test ./internal/supervisor/...`

**Result:** ✅ Test passes

#### Step 3: Refactor

Add window logic:

```go
func (rt *RestartTracker) RecordRestart() {
    rt.restarts = append(rt.restarts, time.Now())

    // Remove restarts outside window
    cutoff := time.Now().Add(-time.Duration(rt.windowSeconds) * time.Second)
    filtered := []time.Time{}
    for _, t := range rt.restarts {
        if t.After(cutoff) {
            filtered = append(filtered, t)
        }
    }
    rt.restarts = filtered
}
```

**Run:** `go test ./internal/supervisor/...`

**Result:** ✅ Tests still pass

#### Step 4: Add More Tests

```go
func TestCrashLoopWindow(t *testing.T) {
    tracker := NewRestartTracker(3, 2)  // max=3, window=2s

    tracker.RecordRestart()
    time.Sleep(1 * time.Second)
    tracker.RecordRestart()
    time.Sleep(1 * time.Second)
    tracker.RecordRestart()

    assert.False(t, tracker.InCrashLoop())  // Only 1 restart in last 2s window

    tracker.RecordRestart()
    tracker.RecordRestart()
    tracker.RecordRestart()

    assert.True(t, tracker.InCrashLoop())  // 3 restarts in last 2s
}
```

---

**End of Testing Strategy**
