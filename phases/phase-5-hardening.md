# Phase 5: Hardening (Week 5)

**Goal:** Chaos tests, benchmarks, real-world validation, and documentation.

---

## Tasks

### 1. Chaos Tests

**Files:**
- `test/fixtures/slow_server.py` - 30s startup server
- `test/fixtures/chatty_server.py` - Log spam server
- `test/integration/chaos_test.go` - Chaos scenarios

**Scenarios:**

#### slow_server.py
```python
#!/usr/bin/env python3
import json
import sys
import time

# Simulate slow startup
time.sleep(30)

# Then normal echo server
def main():
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

#### chatty_server.py
```python
#!/usr/bin/env python3
import json
import sys
import time

def main():
    # Send initialize response
    line = sys.stdin.readline()
    req = json.loads(line)
    resp = {
        "jsonrpc": "2.0",
        "id": req["id"],
        "result": {"protocolVersion": "1.0"}
    }
    print(json.dumps(resp), flush=True)

    # Spam logs
    while True:
        print("DEBUG: " + "X" * 1000, file=sys.stderr, flush=True)
        time.sleep(0.01)  # 100 logs/second

if __name__ == "__main__":
    main()
```

**Tests:**
- Slow startup (30s) with queued requests
- Chatty server with rate limiting
- Intermittent corruption (logs mixed with JSON-RPC)
- SIGTERM hang (server ignores signal)
- Mass file changes (git checkout simulation)
- Subscription during restart
- Concurrent requests during restart

---

### 2. Benchmark Tests

**Files:**
- `test/benchmarks/proxy_test.go` - Proxy overhead
- `test/benchmarks/restart_test.go` - Restart speed
- `test/benchmarks/memory_test.go` - Memory leak detection

**Benchmarks:**

#### Proxy Overhead
```go
func BenchmarkProxyOverhead(b *testing.B) {
    hydra := startHydra(b, "test/fixtures/echo_server.py")
    defer hydra.Stop()

    msg := []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`)

    var latencies []time.Duration

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        hydra.SendRequest(string(msg))
        latency := time.Since(start)
        latencies = append(latencies, latency)
    }

    // Calculate percentiles
    sort.Slice(latencies, func(i, j int) bool {
        return latencies[i] < latencies[j]
    })

    p50 := latencies[len(latencies)/2]
    p99 := latencies[len(latencies)*99/100]

    b.Logf("p50: %v", p50)
    b.Logf("p99: %v", p99)

    if p50 > 50*time.Millisecond {
        b.Fatalf("p50 latency %v > 50ms", p50)
    }
    if p99 > 200*time.Millisecond {
        b.Fatalf("p99 latency %v > 200ms", p99)
    }
}
```

#### Restart Speed
```go
func BenchmarkRestartSpeed(b *testing.B) {
    hydra := startHydra(b, "test/fixtures/echo_server.py")
    defer hydra.Stop()

    var durations []time.Duration

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        hydra.Restart()
        waitForState(b, hydra, StateRunning, 10*time.Second)
        duration := time.Since(start)
        durations = append(durations, duration)
    }

    // Calculate percentiles
    sort.Slice(durations, func(i, j int) bool {
        return durations[i] < durations[j]
    })

    p50 := durations[len(durations)/2]
    p99 := durations[len(durations)*99/100]

    b.Logf("p50: %v", p50)
    b.Logf("p99: %v", p99)

    if p50 > 500*time.Millisecond {
        b.Fatalf("p50 restart time %v > 500ms", p50)
    }
    if p99 > 2*time.Second {
        b.Fatalf("p99 restart time %v > 2s", p99)
    }
}
```

#### Memory Leak Detection
```go
func TestMemoryLeak(t *testing.T) {
    hydra := startHydra(t, "test/fixtures/echo_server.py")
    defer hydra.Stop()

    // Measure initial RSS
    initialRSS := getCurrentRSS()

    // Trigger 1000 restarts
    for i := 0; i < 1000; i++ {
        hydra.Restart()
        waitForState(t, hydra, StateRunning, 5*time.Second)
        time.Sleep(100 * time.Millisecond)
    }

    // Measure final RSS
    finalRSS := getCurrentRSS()

    // Assert memory increase < 100MB
    increase := finalRSS - initialRSS
    maxIncrease := 100 * 1024 * 1024  // 100MB

    if increase > maxIncrease {
        t.Fatalf("Memory leak detected: %d bytes (> %d)", increase, maxIncrease)
    }
}
```

**Performance Targets:**
- Proxy latency p50 < 50ms
- Proxy latency p99 < 200ms
- Restart time p50 < 500ms
- Restart time p99 < 2s
- Memory after 1000 restarts < 100MB

---

### 3. Real-World Integration Tests

**Tests:**

#### Claude Desktop Integration
```go
func TestClaudeDesktopIntegration(t *testing.T) {
    if os.Getenv("CI") == "true" {
        t.Skip("Requires Claude Desktop installed")
    }

    // Find Claude config
    claudeConfig := "~/.config/claude/claude_desktop_config.json"

    // Backup original
    backupFile(claudeConfig)
    defer restoreFile(claudeConfig)

    // Run hydra init
    runCommand("hydra", "init", "--client", "claude")

    // Start Claude Desktop (via AppleScript on macOS)
    startClaudeDesktop()
    defer stopClaudeDesktop()

    // Wait for connection
    time.Sleep(5 * time.Second)

    // Trigger 10 restarts
    for i := 0; i < 10; i++ {
        modifyServerFile()  // Trigger file watcher
        time.Sleep(2 * time.Second)
    }

    // Verify Claude session still alive
    // (Check for connection, tool availability, etc.)
}
```

#### Python/Node/Go Server Tests
```go
func TestPythonServer(t *testing.T) {
    // Test with real Python MCP server
}

func TestNodeServer(t *testing.T) {
    // Test with real Node MCP server
}

func TestGoServer(t *testing.T) {
    // Test with real Go MCP server
}
```

---

### 4. Documentation

**Files to Create:**

#### README.md
```markdown
# Hydra

Fault-tolerant supervisor and proxy for MCP servers.

## Quickstart

### Install
```bash
brew install hydra  # macOS
# or
go install github.com/yourorg/hydra@latest
```

### Setup
```bash
hydra init --client claude
# Follow prompts to select servers

# Restart Claude Desktop
```

### Usage
```bash
# Start server
hydra run --name my-server

# View logs
hydra logs my-server --follow

# Check status
hydra status my-server
```

## Features
- Crash recovery
- Hot-reload on file changes
- Session continuity
- Request queueing during restarts
- Tool injection (hydra_restart, hydra_status, hydra_logs)
- Traffic recording for debugging

## Documentation
- [Configuration Reference](docs/CONFIGURATION.md)
- [CLI Reference](docs/CLI_REFERENCE.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Security](docs/SECURITY.md)

## License
MIT
```

#### CHANGELOG.md
```markdown
# Changelog

## [1.0.0] - 2026-XX-XX

### Added
- Initial release
- MCP server supervision
- Crash recovery
- Hot-reload on file changes
- Injectable tools (hydra_*)
- Traffic recorder
- CLI commands (run, init, add, list, remove, logs, status, restart, recover)
```

#### CONTRIBUTING.md
```markdown
# Contributing

## Development Setup

```bash
git clone github.com/yourorg/hydra
cd hydra
make test
```

## Testing
```bash
make test       # Unit tests
make test-race  # Race detector
make lint       # Linting
```

## Submitting PRs
1. Write tests
2. Ensure `make test` passes
3. Ensure `make lint` passes
4. Update CHANGELOG.md
5. Submit PR
```

---

### 5. CI/CD Pipeline

**File:** `.github/workflows/test.yml`

```yaml
name: Test
on: [push, pull_request]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        go: ['1.23']

    runs-on: ${{ matrix.os }}

    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: ${{ matrix.go }}

      - name: Install Dependencies
        run: go mod download

      - name: Lint
        run: |
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
          golangci-lint run

      - name: Unit Tests
        run: go test -race -coverprofile=coverage.out ./...

      - name: Coverage Check
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $coverage%"
          if (( $(echo "$coverage < 80" | bc -l) )); then
            echo "Coverage below 80%"
            exit 1
          fi

      - name: Integration Tests
        run: go test -race ./test/integration/...

      - name: Benchmarks
        run: go test -bench=. -benchmem ./test/benchmarks/...

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Build
        run: |
          GOOS=darwin GOARCH=amd64 go build -o bin/hydra-darwin-amd64 cmd/hydra/main.go
          GOOS=darwin GOARCH=arm64 go build -o bin/hydra-darwin-arm64 cmd/hydra/main.go
          GOOS=linux GOARCH=amd64 go build -o bin/hydra-linux-amd64 cmd/hydra/main.go
          GOOS=windows GOARCH=amd64 go build -o bin/hydra-windows-amd64.exe cmd/hydra/main.go

      - uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: bin/
```

---

### 6. Homebrew Formula (macOS)

**File:** `homebrew/hydra.rb`

```ruby
class Hydra < Formula
  desc "Fault-tolerant supervisor for MCP servers"
  homepage "https://github.com/yourorg/hydra"
  url "https://github.com/yourorg/hydra/archive/v1.0.0.tar.gz"
  sha256 "..."
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"hydra", "cmd/hydra/main.go"
  end

  test do
    system bin/"hydra", "--version"
  end
end
```

---

## Definition of Done (Phase 5)

- [ ] All chaos tests pass
- [ ] All benchmarks meet targets
- [ ] Real Claude Desktop integration works
- [ ] Works with Python/Node/Go servers
- [ ] README.md complete
- [ ] All docs updated
- [ ] CI/CD pipeline configured
- [ ] Binaries build for all platforms
- [ ] Homebrew formula created
- [ ] 90%+ overall test coverage
- [ ] No known bugs
- [ ] Ready for v1.0 release

---

## Files Created (Phase 5)

```
test/fixtures/slow_server.py
test/fixtures/chatty_server.py
test/integration/chaos_test.go
test/benchmarks/proxy_test.go
test/benchmarks/restart_test.go
test/benchmarks/memory_test.go
README.md
CHANGELOG.md
CONTRIBUTING.md
LICENSE
.github/workflows/test.yml
.github/workflows/release.yml
homebrew/hydra.rb
```

---

## Estimated Time

**5-7 days**

---

**End of Phase 5 - Project Complete!**
