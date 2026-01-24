# Hydra PRD: Core Specification

## Purpose
Hydra is a fault-tolerant supervisor and proxy for MCP servers. It sits between AI clients and MCP servers, providing crash recovery, hot-reload, and session continuity via stdio interception. When the child server crashes or restarts, the AI client session remains alive.

---

## Architecture Decision: 1:1 Model

**One Hydra instance per MCP server** (not multiplexed).

**Rationale:**
- MCP protocol assumes 1 stdio connection = 1 server
- Tool namespacing breaks with multiplexing
- Multiple lightweight instances (< 10MB each) is acceptable

**Topology:**
```
AI Client (Claude Desktop)
  ├─ stdio → Hydra Instance #1 → Python MCP Server
  ├─ stdio → Hydra Instance #2 → Node MCP Server
  └─ stdio → Hydra Instance #3 → Go MCP Server
```

---

## Configuration System

### Two-Tier Model

**Tier 1: Global Registry** (Required)
- Location: `~/.hydra/config.json`
- Single source of truth for all servers
- See `docs/CONFIGURATION.md` for complete schema

**Tier 2: Local Override** (Optional)
- Location: `$CWD/hydra.json`
- Per-project overrides (watch paths, behavior)

**Discovery:** `hydra run --name my-server`
```
1. Load ~/.hydra/config.json (REQUIRED)
2. Find server entry by name
3. Load ./hydra.json if present (OPTIONAL)
4. Merge: defaults < registry < local overrides
5. Validate and start
```

---

## State Machine

### States

```
STOPPED → STARTING → RUNNING → RESTARTING → FAILED
                                    ↓            ↓
                                 (success)   (recover)
                                    ↓            ↓
                                 RUNNING ← ← STOPPED
```

### State Definitions

| State | Description | Client Requests |
|-------|-------------|-----------------|
| STOPPED | No child process | Error response |
| STARTING | Child spawned, waiting for initialize | Queued, replayed after RUNNING |
| RUNNING | Child healthy, initialized | Forwarded immediately |
| RESTARTING | Child terminated, new starting | Queued (max 100, 30s TTL) |
| FAILED | Crash loop detected | Error response |

### Transitions

| From | Event | To | Action |
|------|-------|----|----|
| STOPPED | `hydra run` | STARTING | Spawn child |
| STARTING | initialize response | RUNNING | Replay state |
| STARTING | Timeout (10s) | FAILED | Kill child |
| RUNNING | File change / crash | RESTARTING | Graceful shutdown, spawn new |
| RESTARTING | initialize response | RUNNING | Replay state, drain queue |
| RESTARTING | max_restarts exceeded | FAILED | Send crash loop notification |
| FAILED | `hydra recover` | STOPPED | Reset counter |

**Details:** See `docs/ARCHITECTURE.md`

---

## Core Interfaces

All components MUST use interface-first pattern:

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

**Required Interfaces:**
- Transport (stdio communication)
- Sanitizer (STDIO pollution filter)
- Supervisor (process lifecycle)
- StateStore (session state cache)
- Watcher (file changes)
- Recorder (traffic debugging)
- Proxy (orchestrator)

**Details:** See `docs/ARCHITECTURE.md`

---

## Go Development Standards

### Banned Practices

1. **`fmt.Println` / `fmt.Printf`** in production
   - Corrupts JSON-RPC stdout stream
   - Exception: `fmt.Errorf` allowed
   - Use `internal/logger` (writes to stderr)

2. **Global variables**
   - Breaks testability
   - Use dependency injection

3. **Direct struct initialization** (for components)
   - Use `New*()` constructors

4. **Unchecked errors**
   - Enforced by `golangci-lint`

### Required Practices

1. Interface-first pattern (all components)
2. Mock implementations (all interfaces)
3. 80%+ test coverage (95%+ for critical paths)
4. Error wrapping: `fmt.Errorf("context: %w", err)`
5. Goroutine lifecycle management (context.Context)
6. Race-free code (`-race` flag in CI)

---

## Security Model

### Threat Boundaries

| Threat | In Scope | Mitigation |
|--------|----------|------------|
| Malicious server code | ❌ No | Out of scope (if server is malicious, user is compromised) |
| Config injection | ✅ Yes | Schema validation, no shell expansion |
| STDIO injection | ✅ Yes | Sanitizer validates JSON-RPC |
| Secret leakage | ✅ Yes | Redaction patterns |
| DoS (large payloads) | ✅ Yes | 50KB size limit |
| Zombie processes | ✅ Yes | Tree kill (SysProcAttr.Setpgid) |
| Pre-restart hook hangs | ✅ Yes | Timeout + SIGKILL |

**Details:** See `docs/SECURITY.md`

---

## Injectable Tools

**Reserved Namespace:** `hydra_*`

All Hydra tools start with `hydra_`. If child server exposes `hydra_*` tools, Hydra refuses to start (namespace collision).

**Built-in Tools:**
- `hydra_restart` - Manual restart
- `hydra_status` - Get supervisor status
- `hydra_logs` - View child stderr logs (50-line buffer, default 20 lines returned)
- `hydra_force_restart` - Override crash loop protection

**Details:** See `docs/INJECTABLE_TOOLS.md`

---

## Token Safety: "Wallet Guard"

Hydra protects AI agents from token bombs:

1. **Log Truncation:** 1000 chars max per log message
2. **Log Buffer:** 50-line circular buffer (in-memory only, no persistence)
3. **Payload Limiting:** 50KB max per JSON-RPC response
4. **Rate Limiting:** 10 logs/second max from child

**Applied to:** Child logs, tool outputs, error messages

---

## Project Structure

```
/cmd/hydra/main.go           # Entry point (< 100 lines)
/internal
  /config                    # Config load, merge, validate
  /transport                 # Stdio + protocol detection
  /sanitizer                 # STDIO pollution filter
  /supervisor                # Process lifecycle
  /statestore                # State replay
  /watcher                   # File watching + debounce
  /recorder                  # Traffic debugging
  /proxy                     # Orchestrator
  /injectable                # hydra_* tools
  /security                  # Redaction, rate limiting
  /cli                       # CLI commands
  /logger                    # Zerolog wrapper
/test
  /fixtures                  # Test servers
  /integration               # Integration tests
  /unit                      # Unit tests
```

**Rule:** No file > 250 lines (300 max for rare complex cases, generated code exempt)

---

## Performance Targets

| Metric | Target |
|--------|--------|
| Proxy latency (p50) | < 50ms |
| Proxy latency (p99) | < 200ms |
| Restart time (p50) | < 500ms |
| Restart time (p99) | < 2s |
| Memory (after 1000 restarts) | < 100MB RSS |
| CPU (idle) | < 1% |
| File watch latency | < 100ms |

**Enforced in CI** - Performance regression = CI failure

---

## Definition of Done

### Code Quality
- [ ] 80%+ test coverage (90%+ for proxy/supervisor)
- [ ] Tests pass with `-race` flag
- [ ] `golangci-lint` passes (zero warnings)
- [ ] All interfaces have mocks
- [ ] No `fmt.Println` in production code
- [ ] No global variables

### Functionality
- [ ] All CLI commands work
- [ ] All injectable tools work
- [ ] State machine transitions tested
- [ ] Crash loop detection works
- [ ] Subscription resurrection works
- [ ] Request queueing works
- [ ] File watch debouncing works
- [ ] Pre-restart hooks work
- [ ] Secret redaction works
- [ ] Payload size limiting works

### Performance
- [ ] All benchmark targets met
- [ ] No memory leaks

### Integration
- [ ] Works with Claude Desktop
- [ ] Works with Python/Node/Go MCP servers
- [ ] Chaos tests pass

### Documentation
- [ ] README with quickstart
- [ ] All docs complete
- [ ] CLI reference complete

---

## Implementation Phases

**See `/phases` folder for detailed week-by-week plans:**

1. **Phase 1: Foundation** - Config, transport, sanitizer
2. **Phase 2: Core Logic** - Supervisor, statestore, watcher
3. **Phase 3: Orchestration** - Proxy, tools, recorder
4. **Phase 4: CLI** - Commands, bootstrap logic
5. **Phase 5: Hardening** - Chaos tests, benchmarks, docs

---

## Related Documentation

- `docs/ARCHITECTURE.md` - Detailed state machines, algorithms
- `docs/CONFIGURATION.md` - Complete config schema
- `docs/CLI_REFERENCE.md` - All CLI commands
- `docs/SECURITY.md` - Threat model, redaction
- `docs/INJECTABLE_TOOLS.md` - Tool specifications
- `docs/testing/TESTING_STRATEGY.md` - TDD approach
- `docs/testing/TEST_SCENARIOS.md` - Test cases
- `phases/phase-*.md` - Implementation plans

---

## AI Agent Implementation Notes

1. **Start with interfaces** - Define all interfaces before implementation
2. **TDD mandatory** - Write test, watch fail, implement, refactor
3. **Files < 250 lines** - Split large files into subpackages (300 max rare exception)
4. **Reference this PRD** for core rules (read frequently)
5. **Reference specialized docs** for implementation details (read as-needed)
6. **When in doubt, ask** - Don't guess behavior

---

**End of Core PRD** - See `/docs` and `/phases` for implementation details.
