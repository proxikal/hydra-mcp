# Hydra: MCP Hot Reload Supervisor

## Problem Statement
AI agents (Claude, Gemini, Codex) developing MCP servers lose session context when the child MCP server crashes. Current tools (`mcp-hot-reload`) are protocol-unaware pipes: crash = connection death = lost session.

## Core Architecture

```
┌─────────────┐ stdio  ┌─────────────┐ stdio  ┌─────────────┐
│             │◄──────►│             │◄──────►│             │
│  AI Agent   │        │    Hydra    │        │ MCP Server  │
│  (Claude)   │        │   (Proxy)   │        │ (Dev Code)  │
│             │        │             │        │             │
└─────────────┘        └─────────────┘        └─────────────┘
     NEVER                 ALWAYS                  CAN DIE
     DIES                  ALIVE                   & RESTART
```

**Guarantee:** AI ←→ Hydra connection never breaks. Hydra supervises child MCP server lifecycle.

## Design Principles

1. **Session Persistence:** Crashes are recoverable tool errors, not connection failures
2. **AI-Readable Errors:** Error messages are structured JSON-RPC responses the AI can parse and fix
3. **Transparent Proxy:** Normal operations pass through unchanged
4. **No Infinite Loops:** Progressive backoff prevents AI death spirals
5. **Atomic File Operations:** Only restart on complete file writes

## State Machine

### States
- `HEALTHY`: Child server running, proxy mode active
- `RESTARTING`: Child server spawning, requests queued/rejected
- `CRASHED`: Child server dead, sending error responses

### Transitions

```
HEALTHY → file change → validate syntax → RESTARTING → spawn success → HEALTHY
HEALTHY → child crash → CRASHED → file change → RESTARTING
RESTARTING → spawn failure → CRASHED
```

## Crash Loop Prevention

**Progressive Backoff with Change Detection:**

### Crash #1
- Immediate restart (no delay)
- Error response:
  ```json
  {
    "isError": true,
    "content": [{
      "type": "text",
      "text": "⚠️ SERVER CRASHED: <error_type> in <file>:<line>\nConnection remains open. Fix and save to retry."
    }]
  }
  ```

### Crash #2
- 2-3 second delay before restart
- Enhanced error with history:
  ```json
  {
    "isError": true,
    "content": [{
      "type": "text",
      "text": "⚠️ CRASH #2: <error_type> in <file>:<line>\nPrevious crash: <error_type> in <file>:<line>\nConnection remains open. Fix and save to retry."
    }]
  }
  ```

### Crash #3+
- 5-10 second delay
- **Require file content changed** since last crash (prevents identical broken code loops)
- Error with full history:
  ```json
  {
    "isError": true,
    "content": [{
      "type": "text",
      "text": "⚠️ CRASH #3: <error_type> in <file>:<line>\n\nCrash history (last 5):\n1. <error> (0s ago)\n2. <error> (5s ago)\n3. <error> (12s ago)\n\nSuggestion: Consider a different approach.\nConnection remains open. Make changes and save to retry."
    }]
  }
  ```

**No hard blocks.** AI can always iterate, but delays + context nudge away from loops.

## Re-Initialization Strategy

When child server restarts, it needs MCP initialization handshake (`initialize` → `initialized`).

**Approach: Transparent Re-Init with Success Notification**

1. Hydra caches original `initialize` request from AI
2. On restart, Hydra replays initialization to new child process
3. Child responds with capabilities
4. Hydra sends success notification to AI:
   ```json
   {
     "type": "text",
     "text": "✓ Server restarted successfully. Tools: [list_files, search_code, run_command]"
   }
   ```

**Rationale:** AI needs explicit success signals, not just silence. Tool list confirms available capabilities.

## File Integrity & Restart Triggers

**Pre-Flight Validation:**
- On file change, run syntax validation BEFORE restart
  - Python: `python -m py_compile <file>` or `ast.parse()`
  - Node: `node --check <file>` or `esbuild --check`
  - Go: `go build -o /dev/null`
- If validation fails → immediate error, skip restart
- If validation passes → proceed with restart

**Atomic File Writes:**
- Debounce fsnotify events (editors write multiple times)
- Only trigger restart on complete write (close event)
- Prevents restarts on partial/corrupted file states

**Restart Timing:**
- Target < 2s for simple servers
- Accept 5-10s for heavy servers (Python ML libs, etc.)
- No artificial delays - restart as fast as possible

## Request Handling During Restart

**Realistic Constraint:** Restarts can take 5-10s for Python servers with heavy dependencies.

**Strategy: Fast Fail with Clear Error**

When request arrives during `RESTARTING` state:
- If restart in progress < 3s → queue request, wait for completion
- If restart in progress > 3s → immediate error response:
  ```json
  {
    "isError": true,
    "content": [{
      "type": "text",
      "text": "Server is restarting. Retry in 2 seconds."
    }]
  }
  ```

**No complex buffering.** Fast feedback > silent waiting.

## Transparent Proxy Behavior

**During HEALTHY state:**
- Parse incoming JSON-RPC to track state
- Forward all bytes unchanged to child server
- Forward all responses unchanged to AI
- Zero modification, zero latency overhead

**Operations that must work identically:**
- `initialize` - forwarded
- `tools/list` - forwarded
- `tools/call` - forwarded
- `resources/list` - forwarded
- `resources/read` - forwarded
- `prompts/list` - forwarded
- Custom protocol extensions - forwarded

**Only intercept during CRASHED/RESTARTING states.**

## Traffic Inspection (Future Phase)

Log all JSON-RPC traffic to `.hydra/traffic.log`:
```
→ [2025-01-21 16:30:45] tools/call
  {"name": "search_code", "arguments": {"query": "main"}}

← [2025-01-21 16:30:46] result
  {"results": [...]}
```

Benefit: Developers see exactly what AI sent, what server returned.

## Auto-Detection (Future Phase)

Detect project type and configure automatically:
- `pyproject.toml` → Python mode, watch `**/*.py`
- `package.json` → Node mode, watch `**/*.js`, `**/*.ts`
- `go.mod` → Go mode, watch `**/*.go`

If multiple present → explicit configuration required.

## Success Criteria

**For this to be better than `mcp-hot-reload`:**
1. AI sessions survive crashes ✅
2. AI receives readable error messages ✅
3. AI gets explicit success confirmations ✅
4. Crash loops are prevented ✅
5. Normal operations are transparent ✅

**Excellence Metric:** AI can iterate on broken MCP server 10+ times in single session without manual intervention.

## Implementation Stack

- **Language:** Go
- **Concurrency:** Goroutines for supervisor, proxy, watcher
- **File Watching:** `fsnotify`
- **Process Management:** `os/exec` with signal handling
- **JSON-RPC Parsing:** Streaming parser for stdin/stdout

## Current Phase

**Phase 1 Complete:** Basic process wrapping + fsnotify
**Phase 2 (Next):** JSON-RPC proxy + crash recovery + re-initialization
**Phase 3:** Traffic inspection + CLI polish
