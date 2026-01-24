# Log Optimization: 500 → 50 Lines

**Date:** 2026-01-24
**Reason:** Prevent token waste on AI log consumption

## Changes Made

### 1. Log Buffer Reduced
**Before:**
- Buffer size: 500 lines
- Default return: 50 lines
- Max return: 500 lines

**After:**
- Buffer size: 50 lines
- Default return: 20 lines
- Max return: 50 lines

**Impact:** 90% reduction in max token cost (25K-50K → 2.5K-5K tokens)

### 2. Files Modified

**Code:**
- `internal/injectable/logs.go` - NewLogBuffer default: 500 → 50
- `internal/injectable/tools.go` - MaxLogLines default: 500 → 50
- `internal/injectable/logs.go` - handleLogs default: 50 → 20 lines
- `internal/watcher/fswatch_paths.go` - Reduced directory watch spam (log only root, not every subdir)

**Tests:**
- `internal/injectable/logs_test.go` - Updated test expectations and names

**Documentation:**
- `PRD.md` - Added log buffer details to Injectable Tools and Token Safety sections
- `~/.claude/skills/CerberusBeta/tools/hydra.md` - Updated limits and costs

### 3. Logging Audit Results

**Supervisor logging (process.go):**
- 4 Error logs only (critical failures)
- 0 Info/Debug/Warn logs
- ✅ Clean and minimal

**Watcher logging (fswatch_paths.go):**
- Fixed: Was logging every subdirectory added (potential 100+ line spam)
- Now: Logs only root directory per watch path
- ✅ No more spam

**Overall stats:**
- Core components (proxy/supervisor/transport): 10 Error logs, 4 Info/Warn logs total
- No verbose logging in production code
- ✅ AI-friendly logging

### 4. Architecture Verification

**Log persistence:** None
- Logs stored in circular buffer (in-memory only)
- Buffer cleared on restart
- No accumulation across sessions
- ✅ No 5K token burns from old projects

**Log flow:**
- Child stderr → Circular buffer (50 lines max)
- `hydra_logs` tool → Returns last N lines (default 20, max 50)
- Secrets auto-redacted
- ✅ Token-efficient

### 5. Test Results

All tests pass with race detector:
```
ok  	internal/injectable	1.762s
ok  	internal/watcher	2.878s
```

## Rationale

**Why 50 lines is plenty:**
- Typical crash: 10-15 lines (stack trace + context)
- Crash loop (10 crashes): Pattern visible in 20-30 lines
- If diagnosis needs >50 lines, fix is improving Hydra's logging, not increasing buffer

**Token savings:**
- Old max (500 lines): 25K-50K tokens
- New max (50 lines): 2.5K-5K tokens
- **90% reduction** in worst-case token cost

**Philosophy:**
Well-behaved supervisor = quiet logs. Hydra should be precise, not verbose.
