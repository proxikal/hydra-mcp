# Contributing to Hydra

Thank you for your interest in contributing to Hydra! This document provides guidelines and standards for development.

## Development Setup

### Prerequisites
- Go 1.23 or higher
- Python 3.8+ (for test fixtures)
- Git

### Clone and Build
```bash
git clone https://github.com/proxikal/hydra.git
cd hydra
go mod download
go build -o bin/hydra cmd/hydra/main.go
```

### Run Tests
```bash
# Unit tests
go test ./...

# Unit tests with race detector
go test -race ./...

# Integration tests
go test ./test/integration/...

# Benchmarks
go test -bench=. ./test/benchmarks/...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Development Standards

### Code Quality Requirements

All contributions MUST meet these standards:

#### 1. **No `fmt.Println` in Production Code**
- Corrupts JSON-RPC stdout stream
- Use `internal/logger` (writes to stderr)
- Exception: `fmt.Errorf` is allowed for error wrapping

```go
// ❌ WRONG
fmt.Println("Debug info")

// ✅ CORRECT
logger.Debug("Debug info")
```

#### 2. **No Global Variables**
- Breaks testability and causes race conditions
- Use dependency injection via constructors

```go
// ❌ WRONG
var globalLogger *zerolog.Logger

// ✅ CORRECT
type Manager struct {
    logger *zerolog.Logger
}
func NewManager(logger *zerolog.Logger) *Manager {
    return &Manager{logger: logger}
}
```

#### 3. **Interface-First Pattern**
All components must follow this pattern:

```go
// 1. Interface (public)
type Manager interface {
    Start() error
    Stop() error
}

// 2. Struct (private)
type manager struct {
    cmd    *exec.Cmd
    logger *zerolog.Logger
}

// 3. Constructor (public)
func NewManager(logger *zerolog.Logger) Manager {
    return &manager{logger: logger}
}
```

#### 4. **File Size Limit**
- No file > 250 lines (300 max for rare complexity, generated code exempt)
- Split large files by concern, not alphabetically
- Examples: `forwarding.go`, `recording.go`, `state_updates.go`

#### 5. **Test-Driven Development (TDD)**
1. Write test first (watch it fail)
2. Implement minimal code to pass
3. Refactor while keeping tests green
4. All interfaces must have mocks (use `mockery`)

#### 6. **Error Handling**
- Check all errors (enforced by `golangci-lint`)
- Wrap errors with context: `fmt.Errorf("context: %w", err)`

#### 7. **Goroutine Management**
- All goroutines must respect `context.Context`
- No goroutine leaks (use `t.Cleanup()` in tests)

### Testing Requirements

#### Coverage Targets
- Overall: 80%+ test coverage
- Critical paths (proxy, supervisor): 90%+

#### Required Tests
1. **Unit tests**: All public functions
2. **Integration tests**: State machine transitions
3. **Race tests**: Run with `-race` flag
4. **Benchmarks**: Performance-critical paths

#### Test File Naming
```
implementation_test.go       # Core tests + shared helpers
implementation_feature_test.go  # Feature-specific tests (if needed for size)
```

### Code Review Process

#### Before Submitting PR
1. Run `go test -race ./...` (must pass)
2. Run `golangci-lint run` (zero warnings)
3. Run `go test -coverprofile=coverage.out ./...` (check coverage)
4. Update `CHANGELOG.md` under `[Unreleased]`
5. Ensure all files < 250 lines (300 max for rare complexity)
6. Verify no `fmt.Println` in production code

#### PR Guidelines
- One feature/fix per PR
- Clear title and description
- Reference related issues
- Include test coverage
- Update documentation if needed

#### Review Criteria
PRs will be reviewed for:
- Code quality (follows standards above)
- Test coverage (meets targets)
- Documentation (clear, concise)
- Performance (no regressions)
- Security (no vulnerabilities)

## Project Structure

```
/cmd/hydra/main.go           # Entry point
/internal
  /config                    # Configuration loading
  /transport                 # STDIO communication
  /sanitizer                 # STDIO pollution filter
  /supervisor                # Process lifecycle
  /statestore                # State replay
  /watcher                   # File watching
  /recorder                  # Traffic debugging
  /proxy                     # Orchestrator
  /injectable                # hydra_* tools
  /security                  # Redaction, rate limiting
  /cli                       # CLI commands
  /logger                    # Zerolog wrapper
/test
  /fixtures                  # Test servers
  /integration               # Integration tests
  /benchmarks                # Performance tests
/docs                        # Documentation
/phases                      # Implementation plans
```

## Commit Guidelines

### Commit Message Format
```
<type>: <subject>

<body>

<footer>
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `docs`: Documentation changes
- `perf`: Performance improvements
- `chore`: Build/tooling changes

### Example
```
feat: Add hydra_force_restart injectable tool

Adds new injectable tool that allows AI agents to override
crash loop protection and force a server restart.

Closes #42
```

## Getting Help

- **Issues**: [GitHub Issues](https://github.com/proxikal/hydra/issues)
- **Discussions**: [GitHub Discussions](https://github.com/proxikal/hydra/discussions)
- **Documentation**: See `/docs` folder

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
