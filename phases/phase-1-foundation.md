# Phase 1: Foundation (Week 1)

**Goal:** Establish project structure, core packages, and basic building blocks.

---

## Tasks

### 1. Project Setup

**Create project structure:**
```bash
go mod init github.com/yourorg/hydra
mkdir -p cmd/hydra internal test/{fixtures,integration,unit,benchmarks}
```

**Install dependencies:**
```bash
go get github.com/spf13/cobra
go get github.com/rs/zerolog
go get github.com/tidwall/gjson
go get github.com/tidwall/sjson
go get github.com/fsnotify/fsnotify
go get github.com/joho/godotenv
go get github.com/sabhiram/go-gitignore
go get github.com/shirou/gopsutil/v3
go get github.com/stretchr/testify
```

**Setup linting:**
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create .golangci.yml
```

**.golangci.yml:**
```yaml
linters:
  enable:
    - errcheck
    - gofmt
    - goimports
    - govet
    - ineffassign
    - staticcheck
    - unused

linters-settings:
  errcheck:
    check-blank: true
```

**Create Makefile:**
```makefile
.PHONY: test test-race lint build

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run

build:
	go build -o bin/hydra cmd/hydra/main.go

mocks:
	mockery --all --dir=internal --output=internal/mocks
```

---

### 2. Config Package

**Files:**
- `internal/config/config.go` - Struct definitions
- `internal/config/loader.go` - Load, merge, validate
- `internal/config/registry.go` - Global registry operations
- `internal/config/envsubst.go` - ${env:VAR} substitution
- `internal/config/config_test.go` - Unit tests

**Interface:**
```go
type Config struct {
    Version  string
    Defaults Defaults
    Servers  map[string]Server
}

type Loader interface {
    Load(path string) (*Config, error)
    Merge(base, override *Config) *Config
    Validate(cfg *Config) error
}
```

**Tests:**
- Load valid config
- Load with missing file → error
- Merge base + override
- Validate required fields
- Environment variable substitution
- Default value substitution

---

### 3. Logger Package

**Files:**
- `internal/logger/logger.go` - Zerolog wrapper

**Requirements:**
- Writes ONLY to stderr (never stdout)
- Configurable log level (debug, info, warn, error)
- Structured logging (key-value pairs)

**Interface:**
```go
type Logger interface {
    Debug(msg string, keysAndValues ...interface{})
    Info(msg string, keysAndValues ...interface{})
    Warn(msg string, keysAndValues ...interface{})
    Error(msg string, keysAndValues ...interface{})
}

func New(level string) Logger
```

**Tests:**
- Verify output goes to stderr
- Verify log levels work
- Verify structured fields

---

### 4. Transport Package

**Files:**
- `internal/transport/transport.go` - Interface
- `internal/transport/stdio.go` - Stdio implementation
- `internal/transport/protocol.go` - Protocol detection
- `internal/transport/framer.go` - NDJSON/LSP framing
- `internal/transport/transport_test.go` - Tests

**Interface:**
```go
type Protocol int

const (
    ProtocolNDJSON Protocol = iota
    ProtocolLSP
)

type Transport interface {
    Read() ([]byte, error)
    Write([]byte) error
    Close() error
    DetectProtocol(timeout time.Duration) (Protocol, error)
}
```

**Tests:**
- Read NDJSON message
- Read LSP message (with Content-Length header)
- Write NDJSON
- Write LSP
- Protocol auto-detection (NDJSON)
- Protocol auto-detection (LSP)
- Protocol detection timeout

---

### 5. Sanitizer Package

**Files:**
- `internal/sanitizer/sanitizer.go` - Interface
- `internal/sanitizer/classifier.go` - Chunk classification
- `internal/sanitizer/utf8.go` - UTF-8 validation
- `internal/sanitizer/sanitizer_test.go` - Tests

**Interface:**
```go
type ChunkType int

const (
    ChunkEmpty ChunkType = iota
    ChunkJSONRPC
    ChunkPollution
)

type Sanitizer interface {
    Classify(chunk []byte) ChunkType
    ValidateUTF8(data []byte) []byte
}
```

**Tests:**
- Classify valid JSON-RPC request
- Classify valid JSON-RPC response
- Classify pollution (debug log)
- Classify empty line
- Classify valid JSON (not JSON-RPC)
- Validate UTF-8 (valid)
- Validate UTF-8 (invalid) → replace with �

---

## Definition of Done (Phase 1)

- [ ] Project structure created
- [ ] Dependencies installed
- [ ] Makefile works (test, lint, build)
- [ ] golangci-lint passes
- [ ] Config package: 90%+ coverage
- [ ] Logger package: 90%+ coverage
- [ ] Transport package: 90%+ coverage
- [ ] Sanitizer package: 90%+ coverage
- [ ] All tests pass with `-race`
- [ ] No `fmt.Println` in code (except tests)
- [ ] All errors wrapped with context

---

## Files Created (Phase 1)

```
cmd/hydra/main.go
internal/config/config.go
internal/config/loader.go
internal/config/registry.go
internal/config/envsubst.go
internal/config/config_test.go
internal/logger/logger.go
internal/logger/logger_test.go
internal/transport/transport.go
internal/transport/stdio.go
internal/transport/protocol.go
internal/transport/framer.go
internal/transport/transport_test.go
internal/sanitizer/sanitizer.go
internal/sanitizer/classifier.go
internal/sanitizer/utf8.go
internal/sanitizer/sanitizer_test.go
Makefile
.golangci.yml
go.mod
go.sum
```

---

## Estimated Time

**5-7 days** (assuming 4-6 hours/day)
