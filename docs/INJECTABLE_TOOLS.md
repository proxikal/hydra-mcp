# Hydra Injectable Tools

Complete specification for `hydra_*` meta-tools injected into MCP tool list.

---

## Overview

Hydra injects meta-tools into the child server's `tools/list` response, allowing AI agents to:
- Manually restart servers
- View supervisor status
- Retrieve child logs
- Force restart from FAILED state

**Namespace:** All Hydra tools start with `hydra_*` prefix (reserved).

---

## Namespace Collision Detection

### Rule

If child server exposes any tool starting with `hydra_*`, Hydra **refuses to start** (default behavior).

### Configuration

```json
{
  "injectable_tools": {
    "enabled": true,
    "tools": ["hydra_restart", "hydra_logs", "hydra_status", "hydra_force_restart"],
    "on_collision": "error"
  }
}
```

**on_collision Options:**

| Value | Behavior |
|-------|----------|
| `error` | Refuse to start, log error with conflicting tool name (DEFAULT) |
| `warn` | Log warning, disable Hydra tool, allow child tool |
| `disable_hydra_tool` | Silently disable Hydra tool (dangerous) |

### Detection Algorithm

```
1. Child server starts, sends tools/list response
2. Hydra parses tool names from response
3. For each Hydra tool in config.injectable_tools.tools:
     If tool name exists in child tools:
       Execute on_collision action
4. If no collision, merge tool lists
```

**Example Collision:**

```
Child tools: ["search_code", "hydra_restart"]
Hydra tools: ["hydra_restart", "hydra_logs", "hydra_status"]

→ COLLISION on "hydra_restart"

→ Action (on_collision="error"):
   Log error: "Namespace collision: Child server exposes reserved tool 'hydra_restart'"
   Refuse to start (exit with code 1)
```

---

## Tool: `hydra_restart`

### Purpose

Manually restart the child MCP server. Use when server needs restart but file watcher hasn't triggered.

### Definition

```json
{
  "name": "hydra_restart",
  "description": "Manually restart the MCP server supervised by Hydra. Use when server needs restart but file watcher hasn't triggered.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "reason": {
        "type": "string",
        "description": "Optional reason for restart (logged for debugging)"
      }
    }
  }
}
```

### Implementation

```go
func (h *Hydra) HandleRestartTool(params map[string]interface{}) (interface{}, error) {
    reason := params["reason"].(string)

    // Log reason to stderr
    h.logger.Info("Manual restart requested", "reason", reason)

    // Trigger restart (same as file change)
    if err := h.supervisor.Restart(); err != nil {
        return nil, fmt.Errorf("restart failed: %w", err)
    }

    // Wait for RUNNING state
    timeout := time.After(10 * time.Second)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if h.supervisor.State() == StateRunning {
                return map[string]interface{}{
                    "success": true,
                    "message": "Server restarted successfully",
                    "state":   "RUNNING",
                }, nil
            }
        case <-timeout:
            return nil, fmt.Errorf("restart timeout")
        }
    }
}
```

### Response

**Success:**
```json
{
  "success": true,
  "message": "Server restarted successfully",
  "state": "RUNNING"
}
```

**Error (server not running):**
```json
{
  "error": {
    "code": -32000,
    "message": "Cannot restart: server not running"
  }
}
```

**Error (in FAILED state):**
```json
{
  "error": {
    "code": -32000,
    "message": "Cannot restart: server in FAILED state (use hydra_force_restart)"
  }
}
```

---

## Tool: `hydra_status`

### Purpose

Get current status of Hydra supervisor and child server. Use for debugging crashes or restart issues.

### Definition

```json
{
  "name": "hydra_status",
  "description": "Get current status of Hydra supervisor and child server. Use for debugging crashes or restart issues.",
  "inputSchema": {
    "type": "object",
    "properties": {}
  }
}
```

### Implementation

```go
func (h *Hydra) HandleStatusTool(params map[string]interface{}) (interface{}, error) {
    status := h.proxy.Status()

    return map[string]interface{}{
        "state":                  status.State.String(),
        "server_name":            h.config.Name,
        "pid":                    status.PID,
        "uptime_seconds":         int(status.Uptime.Seconds()),
        "restarts_in_window":     status.RestartsInWindow,
        "max_restarts":           h.config.Behavior.MaxRestarts,
        "restart_window_seconds": h.config.Behavior.RestartWindowSeconds,
        "last_restart_reason":    status.LastRestartReason,
        "last_error":             status.LastError,
        "queue_size":             status.QueueSize,
        "can_recover":            status.CanRecover,
    }, nil
}
```

### Response

```json
{
  "state": "RUNNING",
  "server_name": "my-python-server",
  "pid": 12345,
  "uptime_seconds": 3600,
  "restarts_in_window": 2,
  "max_restarts": 10,
  "restart_window_seconds": 60,
  "last_restart_reason": "file_change",
  "last_error": null,
  "queue_size": 0,
  "can_recover": false
}
```

**FAILED State Example:**
```json
{
  "state": "FAILED",
  "server_name": "my-python-server",
  "pid": 0,
  "uptime_seconds": 0,
  "restarts_in_window": 11,
  "max_restarts": 10,
  "restart_window_seconds": 60,
  "last_restart_reason": "crash",
  "last_error": "python: SyntaxError: invalid syntax (server.py, line 42)",
  "queue_size": 0,
  "can_recover": true
}
```

### State Values

- `STOPPED` - No child process running
- `STARTING` - Child spawned, waiting for initialize
- `RUNNING` - Child healthy and initialized
- `RESTARTING` - Child terminated, new process starting
- `FAILED` - Crash loop detected

---

## Tool: `hydra_logs`

### Purpose

Retrieve recent stderr logs from child server. Use for debugging server errors.

### Definition

```json
{
  "name": "hydra_logs",
  "description": "Retrieve recent stderr logs from child server. Use for debugging server errors.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "lines": {
        "type": "number",
        "description": "Number of recent log lines to retrieve (default: 50, max: 500)",
        "minimum": 1,
        "maximum": 500
      }
    }
  }
}
```

### Implementation

```go
type LogBuffer struct {
    lines []string
    mu    sync.Mutex
    max   int  // Max capacity (500)
}

func (lb *LogBuffer) Add(line string) {
    lb.mu.Lock()
    defer lb.mu.Unlock()

    lb.lines = append(lb.lines, line)

    // Keep only last 500 lines
    if len(lb.lines) > lb.max {
        lb.lines = lb.lines[len(lb.lines)-lb.max:]
    }
}

func (h *Hydra) HandleLogsTool(params map[string]interface{}) (interface{}, error) {
    lines := 50  // Default
    if l, ok := params["lines"].(float64); ok {
        lines = int(l)
    }

    // Clamp to [1, 500]
    if lines < 1 {
        lines = 1
    }
    if lines > 500 {
        lines = 500
    }

    // Get last N lines
    recentLogs := h.logBuffer.GetRecent(lines)

    // Redact secrets
    for i, log := range recentLogs {
        recentLogs[i] = h.security.RedactSecrets(log, h.config.Security.RedactPatterns)
    }

    return map[string]interface{}{
        "logs":  recentLogs,
        "count": len(recentLogs),
    }, nil
}
```

### Response

```json
{
  "logs": [
    "[2026-01-21 10:30:45] INFO: Server starting...",
    "[2026-01-21 10:30:46] DEBUG: API_KEY=[REDACTED by Hydra]",
    "[2026-01-21 10:30:47] ERROR: Connection failed"
  ],
  "count": 3
}
```

**Empty Logs:**
```json
{
  "logs": [],
  "count": 0
}
```

---

## Tool: `hydra_force_restart`

### Purpose

Force restart even if in FAILED state (crash loop). Resets restart counter. Use after fixing code.

### Definition

```json
{
  "name": "hydra_force_restart",
  "description": "Force restart even if in FAILED state (crash loop). Resets restart counter. Use after fixing code.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "confirm": {
        "type": "boolean",
        "description": "Must be true to confirm force restart"
      }
    },
    "required": ["confirm"]
  }
}
```

### Security

**Disabled by default.** Must opt-in:

```json
{
  "injectable_tools": {
    "tools": ["hydra_restart", "hydra_status", "hydra_logs", "hydra_force_restart"]
  }
}
```

### Implementation

```go
func (h *Hydra) HandleForceRestartTool(params map[string]interface{}) (interface{}, error) {
    confirm, ok := params["confirm"].(bool)
    if !ok || !confirm {
        return nil, fmt.Errorf("confirm must be true")
    }

    currentState := h.supervisor.State()

    // If FAILED, reset counter and transition to STOPPED
    if currentState == StateFailed {
        h.supervisor.ResetRestartCounter()
        h.stateStore.Clear()
        h.supervisor.SetState(StateStopped)
    }

    // Trigger restart
    if err := h.supervisor.Restart(); err != nil {
        return nil, fmt.Errorf("force restart failed: %w", err)
    }

    // Wait for RUNNING
    timeout := time.After(10 * time.Second)
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            state := h.supervisor.State()
            if state == StateRunning {
                return map[string]interface{}{
                    "success": true,
                    "message": "Server force-restarted successfully",
                    "state":   "RUNNING",
                }, nil
            }
            if state == StateFailed {
                return nil, fmt.Errorf("force restart failed (still in crash loop)")
            }
        case <-timeout:
            return nil, fmt.Errorf("force restart timeout")
        }
    }
}
```

### Response

**Success:**
```json
{
  "success": true,
  "message": "Server force-restarted successfully",
  "state": "RUNNING"
}
```

**Error (confirm not true):**
```json
{
  "error": {
    "code": -32602,
    "message": "Invalid params: confirm must be true"
  }
}
```

**Error (still failing):**
```json
{
  "error": {
    "code": -32000,
    "message": "force restart failed (still in crash loop)"
  }
}
```

---

## Tool Injection Workflow

### 1. Child sends tools/list

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {"name": "search_code", ...},
      {"name": "read_file", ...}
    ]
  }
}
```

### 2. Hydra checks for collisions

```go
childTools := parseToolNames(response)  // ["search_code", "read_file"]
hydraTools := config.InjectableTools.Tools  // ["hydra_restart", "hydra_logs", "hydra_status"]

for _, ht := range hydraTools {
    if contains(childTools, ht) {
        // COLLISION!
        handleCollision(ht, config.InjectableTools.OnCollision)
    }
}
```

### 3. Hydra merges tools

```go
mergedTools := append(response.Result.Tools, hydraToolDefinitions...)
```

### 4. Hydra sends merged tools/list to client

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {"name": "search_code", ...},
      {"name": "read_file", ...},
      {"name": "hydra_restart", ...},
      {"name": "hydra_logs", ...},
      {"name": "hydra_status", ...}
    ]
  }
}
```

### 5. Client calls Hydra tool

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "id": 2,
  "params": {
    "name": "hydra_restart",
    "arguments": {"reason": "Testing"}
  }
}
```

### 6. Hydra intercepts and handles

```go
if strings.HasPrefix(params.Name, "hydra_") {
    // Hydra tool - handle internally
    result := h.handleInjectableTool(params.Name, params.Arguments)
    return result
} else {
    // Child tool - forward to child
    h.child.Write(request)
}
```

---

**End of Injectable Tools Documentation**
