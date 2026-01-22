# Hydra Phase 1-4 Fix TODO

**Generated:** 2026-01-22
**Review Status:** Codex implementation reviewed against PRD and phase requirements

---

## Executive Summary

**Overall Status:** 82% Complete - All core functionality works, but needs cleanup for PRD compliance

- ✅ All tests pass (no failures)
- ✅ Project structure correct
- ✅ All mocks generated
- ❌ 4 files exceed 200-line limit
- ❌ 16 linting errors (errcheck + gofmt)
- ❌ Critical packages under-tested (proxy 67.7%, supervisor 79.8%)
- ❌ CLI and bootstrap packages have 0% test coverage

---

## Critical Issues (Must Fix)

### 1. File Size Violations (PRD Rule: No file > 200 lines)

| File | Lines | Over Limit | Action Required |
|------|-------|------------|-----------------|
| `internal/proxy/proxy.go` | 519 | +259% | Split into subpackages |
| `internal/config/loader.go` | 266 | +33% | Extract functions to separate files |
| `internal/supervisor/process.go` | 257 | +28% | Split process lifecycle logic |
| `internal/recorder/recorder.go` | 205 | +3% | Minor refactor |

**Split Strategy:**
- `proxy.go` → `proxy.go` + `lifecycle.go` + `handlers.go` + `events.go`
- `loader.go` → `loader.go` + `resolver.go` + `validator.go`
- `process.go` → `process.go` + `signals.go` + `treekill.go` + `restart.go`
- `recorder.go` → `recorder.go` + `buffer.go` + `export.go`

---

### 2. Test Coverage Gaps

| Package | Current | Target | Gap | Priority |
|---------|---------|--------|-----|----------|
| **proxy** | 67.7% | 95% | **-27.3%** | CRITICAL |
| **supervisor** | 79.8% | 95% | **-15.2%** | CRITICAL |
| **injectable** | 72.5% | 90% | -17.5% | HIGH |
| **recorder** | 75.5% | 90% | -14.5% | HIGH |
| **config** | 82.6% | 90% | -7.4% | MEDIUM |
| **watcher** | 86.9% | 90% | -3.1% | LOW |
| **sanitizer** | 88.2% | 90% | -1.8% | LOW |
| **cli** | 0% | N/A | N/A | HIGH (no tests exist) |
| **bootstrap** | 0% | N/A | N/A | HIGH (no tests exist) |

**Perfect Coverage (100%):**
- ✅ logger
- ✅ statestore

**Excellent Coverage (90%+):**
- ✅ transport (94.0%)
- ✅ security (97.1%)

---

### 3. Linting Errors (16 total)

**errcheck violations (14):**
```
internal/cli/export.go:87:16 - defer in.Close()
internal/cli/export.go:99:17 - defer out.Close()
internal/cli/list.go:36:14 - fmt.Fprintf
internal/cli/logs.go:108:14 - fmt.Fprintf
internal/watcher/fswatch.go:59:19 - w.watcher.Close()
internal/watcher/watcher_test.go:27:14 - w.Stop()
internal/watcher/watcher_test.go:55:14 - w.Stop()
internal/watcher/watcher_test.go:116:14 - w.Stop()
test/integration/crash_loop_test.go:75:24 - cmd.Process.Kill()
test/integration/queue_test.go:40:24 - cmd.Process.Kill()
test/integration/queue_test.go:183:14 - client.Close()
test/integration/queue_test.go:184:13 - child.Close()
test/integration/restart_test.go:77:24 - cmd.Process.Kill()
test/integration/restart_test.go:115:23 - clientTransport.Close()
test/integration/restart_test.go:171:23 - clientTransport.Close()
```

**gofmt violations (2):**
```
internal/supervisor/process.go:15:1 - Not properly formatted
internal/watcher/fswatch.go:15:1 - Not properly formatted
```

**Fix:** Add error checks or `_ = err` for all unchecked returns

---

### 4. Missing Phase Files

These files were specified in phases but may have functionality embedded elsewhere:

**Phase 1 - Transport:**
- ❌ `internal/transport/protocol.go` - Protocol detection
- ❌ `internal/transport/framer.go` - NDJSON/LSP framing
- Status: Functionality likely in `stdio.go`

**Phase 1 - Sanitizer:**
- ❌ `internal/sanitizer/utf8.go` - UTF-8 validation
- Status: Functionality likely in `classifier.go`

**Phase 2 - Supervisor:**
- ❌ `internal/supervisor/signals.go` - SIGTERM/SIGKILL handling
- ❌ `internal/supervisor/treekill.go` - Cross-platform tree kill
- ❌ `internal/supervisor/restart.go` - Restart counter logic
- Status: All in oversized `process.go` (257 lines)

**Phase 2 - StateStore:**
- ❌ `internal/statestore/initialize.go`
- ❌ `internal/statestore/subscriptions.go`
- Status: Functionality split between `statestore.go` and `store.go`

**Phase 2 - Watcher:**
- ❌ `internal/watcher/debounce.go`
- ❌ `internal/watcher/gitignore.go`
- Status: Functionality in `fswatch.go`

**Phase 3 - Injectable:**
- ❌ `internal/injectable/force.go` - hydra_force_restart implementation

**Phase 3 - Recorder:**
- ❌ `internal/recorder/buffer.go`
- ❌ `internal/recorder/export.go`
- Status: All in oversized `recorder.go` (205 lines)

---

## Phase Completion Summary

### Phase 1: Foundation (85% Complete)
- ✅ Project structure
- ✅ Config package (82.6% coverage)
- ✅ Logger package (100% coverage)
- ✅ Transport package (94.0% coverage)
- ✅ Sanitizer package (88.2% coverage)
- ❌ File splitting needed (loader.go)
- ⚠️ Minor coverage increases needed

**Remaining Work:**
- Split `loader.go` into smaller files
- Add 7.4% coverage to config
- Add 1.8% coverage to sanitizer

### Phase 2: Core Logic (80% Complete)
- ✅ Supervisor functionality works
- ✅ StateStore (100% coverage)
- ✅ Watcher functionality works
- ✅ Security package (97.1% coverage)
- ❌ Supervisor under-tested (79.8% vs 95% target)
- ❌ File splitting needed (process.go)
- ⚠️ Minor coverage increases needed

**Remaining Work:**
- Split `process.go` into signals.go, treekill.go, restart.go
- Add 15.2% coverage to supervisor
- Add 3.1% coverage to watcher

### Phase 3: Orchestration (75% Complete)
- ✅ Proxy functionality works
- ✅ Injectable tools work
- ✅ Recorder works
- ✅ Test fixtures exist
- ✅ Integration tests pass
- ❌ **Proxy critically under-tested (67.7% vs 95% target)**
- ❌ Injectable under-tested (72.5%)
- ❌ Recorder under-tested (75.5%)
- ❌ File splitting needed (proxy.go, recorder.go)
- ❌ Missing hydra_force_restart tool

**Remaining Work:**
- Split `proxy.go` (519 lines → multiple files)
- Split `recorder.go` into buffer.go and export.go
- Add 27.3% coverage to proxy (CRITICAL)
- Add 17.5% coverage to injectable
- Add 14.5% coverage to recorder
- Implement `internal/injectable/force.go`

### Phase 4: CLI (90% Complete Functionally, 0% Tested)
- ✅ All CLI commands implemented
- ✅ Bootstrap logic exists
- ✅ Main entry point clean
- ❌ **Zero test coverage for CLI package**
- ❌ **Zero test coverage for bootstrap package**

**Remaining Work:**
- Create `internal/cli/cli_test.go`
- Create tests for all CLI commands
- Create bootstrap tests

---

## Implementation Plan (Priority Order)

### Step 1: Fix Linting Errors (30 min)
- Run `gofmt -w` on all files
- Add error checks for all unchecked returns
- **Goal:** `golangci-lint run` passes with zero warnings

### Step 2: Split Oversized Files (2-3 hours)
1. **proxy.go (519 → ~120 per file)**
   - Extract event handlers → `handlers.go`
   - Extract lifecycle management → `lifecycle.go`
   - Extract channel management → `events.go`

2. **loader.go (266 → ~90 per file)**
   - Extract validation → `validator.go`
   - Extract resolution logic → `resolver.go`

3. **process.go (257 → ~65 per file)**
   - Extract signal handling → `signals.go`
   - Extract tree kill → `treekill.go`
   - Extract restart logic → `restart.go`

4. **recorder.go (205 → ~70 per file)**
   - Extract buffer logic → `buffer.go`
   - Extract export logic → `export.go`

### Step 3: Add Missing Files (1-2 hours)
- Implement `internal/injectable/force.go`
- Create proper file splits per phase specifications

### Step 4: Increase Test Coverage (4-6 hours)

**Priority 1 (CRITICAL):**
- Proxy: 67.7% → 95% (+27.3%)
  - Test state machine transitions
  - Test panic recovery
  - Test queue overflow
  - Test tool injection

- Supervisor: 79.8% → 95% (+15.2%)
  - Test crash loop detection edge cases
  - Test tree kill scenarios
  - Test signal handling

**Priority 2 (HIGH):**
- Injectable: 72.5% → 90% (+17.5%)
- Recorder: 75.5% → 90% (+14.5%)
- CLI: 0% → 80%+ (create test file)
- Bootstrap: 0% → 80%+ (create test file)

**Priority 3 (MEDIUM/LOW):**
- Config: 82.6% → 90% (+7.4%)
- Watcher: 86.9% → 90% (+3.1%)
- Sanitizer: 88.2% → 90% (+1.8%)

### Step 5: Verify Definition of Done (30 min)
Run checklist from PRD:

**Code Quality:**
- [ ] 80%+ test coverage (90%+ for proxy/supervisor)
- [ ] Tests pass with `-race` flag
- [ ] `golangci-lint` passes (zero warnings)
- [ ] All interfaces have mocks
- [ ] No `fmt.Println` in production code
- [ ] No global variables
- [ ] All files < 200 lines

**Functionality:**
- [ ] All CLI commands work
- [ ] All injectable tools work
- [ ] State machine transitions tested
- [ ] Crash loop detection works
- [ ] Request queueing works
- [ ] File watch debouncing works

**Performance:**
- [ ] All benchmark targets met
- [ ] No memory leaks

---

## Current Test Results

```
ok  	github.com/proxikal/hydra/internal/config	      0.248s	coverage: 82.6%
ok  	github.com/proxikal/hydra/internal/injectable	  0.619s	coverage: 72.5%
ok  	github.com/proxikal/hydra/internal/logger	      0.617s	coverage: 100.0%
ok  	github.com/proxikal/hydra/internal/proxy	      0.763s	coverage: 67.7%
ok  	github.com/proxikal/hydra/internal/recorder	    1.561s	coverage: 75.5%
ok  	github.com/proxikal/hydra/internal/sanitizer	  0.943s	coverage: 88.2%
ok  	github.com/proxikal/hydra/internal/security	    1.918s	coverage: 97.1%
ok  	github.com/proxikal/hydra/internal/statestore	  1.143s	coverage: 100.0%
ok  	github.com/proxikal/hydra/internal/supervisor	  3.332s	coverage: 79.8%
ok  	github.com/proxikal/hydra/internal/transport	  1.789s	coverage: 94.0%
ok  	github.com/proxikal/hydra/internal/watcher	    3.226s	coverage: 86.9%
ok  	github.com/proxikal/hydra/test/integration	    2.644s	coverage: [no statements]

? github.com/proxikal/hydra/cmd/hydra [no test files]
? github.com/proxikal/hydra/internal/bootstrap [no test files]
? github.com/proxikal/hydra/internal/cli [no test files]
```

---

## Notes

- All tests currently pass with no failures
- Integration tests exist and work correctly
- Core functionality is solid - this is cleanup work
- Main blocker for "done" is test coverage on proxy/supervisor
- File splitting will improve maintainability and match phase specs
- CLI tests needed before Phase 4 can be considered complete

---

**End of TODO** - Use this document to track progress across sessions
