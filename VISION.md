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

1. **Session Persistence:** Crashes are recoverable tool errors, not connection failures.
2. **Sanitized Communication:** **CRITICAL:** Child stdout is strictly filtered. Only valid JSON-RPC is passed to the AI. Random `console.log` or stack traces are captured and wrapped as logs, preventing protocol corruption.
3. **AI-Readable Errors:** Error messages are structured JSON-RPC responses the AI can parse and fix.
4. **Optimistic Speed:** Restart immediately on changes; let the crash be the validator.
5. **State Consistency:** Buffer and replay state-altering notifications (`didChange`) so the new server wakes up in sync.

## State Machine

### States
- `HEALTHY`: Child server running, proxy mode active (with sanitization)
- `RESTARTING`: Child server spawning, state notifications buffered
- `CRASHED`: Child server dead, sending error responses

### Transitions

```
HEALTHY → file change → RESTARTING (Optimistic) → spawn success → replay state → HEALTHY
HEALTHY → child crash → CRASHED → file change → RESTARTING
RESTARTING → spawn failure → CRASHED
```

## Stdio Sanitization (The "Pollution" Fix)

**The Problem:** Dev servers are messy. A single `console.log("DB connected")` or raw panic stack trace breaks the JSON-RPC pipe, killing the AI session.

**The Fix:**
- **Stdout:** Hydra intercepts *every* line from the child.
- **Filter:** If line is valid JSON-RPC → Forward to AI.
- **Sanitize:** If line is raw text → Wrap in MCP `notifications/message` (log level: info/debug) and send to AI.
- **Stderr:** Always captured and treated as logs/error context.

**Result:** The AI connection is immune to "noisy" server code.


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


## File Integrity & Restart Triggers

**Optimistic Restart Strategy:**
- **Remove Pre-Flight Checks:** Do not run `go build` or syntax checks before restarting.
- **Action:** On file save, kill the old process and spawn the new one immediately.
- **Validation:** If the new process crashes instantly (exit code != 0 within < 500ms), capture `stderr`.
- **Feedback:** Send the capture `stderr` as the "Build/Syntax Error" to the AI.
- **Benefit:** Zero latency on save. The AI treats a "crash on start" identically to a "syntax error".

**Atomic File Writes:**
- Debounce fsnotify events (editors write multiple times)
- Only trigger restart on complete write (close event)
- Prevents restarts on partial/corrupted file states

**Restart Timing:**
- Target < 2s for simple servers
- Accept 5-10s for heavy servers (Python ML libs, etc.)
- No artificial delays - restart as fast as possible

## Request Handling & State Consistency

**1. State Replay (CRITICAL):**
- **The Issue:** If we restart, the new server loses memory of `didOpen` files or `didChange` edits.
- **The Fix:** Hydra acts as a "State Buffer".
  - **Track:** Maintain a cache of the most recent `initialize` request and any `textDocument/didOpen` or `textDocument/didChange` notifications.
  - **Replay:** On successful spawn, immediately replay these notifications to the new server *before* allowing other traffic.
  - **Result:** The new server "wakes up" with the same file state as the old one.

**2. Request Strategy: Fast Fail**
- **Scenario:** Request arrives during `RESTARTING`.
- **Action:** Return JSON-RPC Error `-32000` (Server Not Initialized) immediately.
  ```json
  {
    "code": -32000,
    "message": "Server is restarting. Please retry in 2 seconds.",
    "data": { "retry_after": 2000 }
  }
  ```
- **Rationale:** Don't buffer ID-based requests (complex to synchronize). Let the AI client handle the retry logic.

## Re-Initialization & Capabilities

**Transparent Re-Init:**
1. Hydra replays cached `initialize` to the new child.
2. Child responds with capabilities.
3. **Capability Drift:** If the new server has different tools, Hydra sends a `notifications/capabilities_changed` (or equivalent) to prompt the AI to refresh its tool list.
4. **Success Notification:**
   ```json
   {
     "type": "text",
     "text": "✓ Server restarted & State replayed. Tools: [list_files, ...]"
   }
   ```

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
