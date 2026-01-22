# Hydra Security Model

Security boundaries, threat model, and mitigation strategies.

---

## Threat Model

### In Scope vs Out of Scope

| Threat | In Scope | Mitigation |
|--------|----------|------------|
| **Malicious server code** | ❌ No | Out of scope. If child MCP server is malicious, user is already compromised. Hydra does NOT sandbox child processes. |
| **Config injection** | ✅ Yes | Schema validation, no shell expansion, argv arrays only |
| **STDIO injection** | ✅ Yes | Sanitizer validates JSON-RPC schema before forwarding |
| **Secret leakage (logs/traffic)** | ✅ Yes | Regex-based redaction, truncation, privacy-first defaults |
| **DoS via large payloads** | ✅ Yes | 50KB size limit per JSON-RPC response |
| **Zombie processes** | ✅ Yes | Tree kill using SysProcAttr.Setpgid (Unix) / job objects (Windows) |
| **Pre-restart hook hangs** | ✅ Yes | Timeout enforcement with SIGKILL escalation |
| **MCP protocol exploits** | ⚠️ Partial | Hydra validates JSON-RPC structure but trusts MCP spec compliance |

---

## Secret Redaction

### Default Patterns

Hydra applies regex-based redaction to:
- Child stderr logs
- Traffic recorder exports
- Error messages sent to client
- `hydra_logs` tool output

**Default Patterns:**
```json
{
  "security": {
    "redact_patterns": [
      "sk-[A-Za-z0-9]{32,}",           // OpenAI API keys
      "API[_-]?KEY",                   // Generic API_KEY / API-KEY
      "password",                      // Case-insensitive password
      "Bearer [A-Za-z0-9._-]+",        // JWT Bearer tokens
      "ghp_[A-Za-z0-9]{36}",           // GitHub personal access tokens
      "xox[baprs]-[A-Za-z0-9-]+"       // Slack tokens
    ],
    "redact_replacement": "[REDACTED by Hydra]"
  }
}
```

### Implementation

```go
func RedactSecrets(content string, patterns []string) string {
    for _, pattern := range patterns {
        re := regexp.MustCompile("(?i)" + pattern)
        content = re.ReplaceAllString(content, "[REDACTED by Hydra]")
    }
    return content
}
```

**Applied to:**
1. **Child stderr** → Before forwarding as MCP log notification
2. **Traffic recorder** → Before export to JSON
3. **Error messages** → Before sending to client
4. **hydra_logs tool** → Before returning to AI agent

---

## Pre-Restart Hook Sandboxing

### Security Constraints

Pre-restart hooks run **arbitrary commands** from config. Security measures:

#### 1. No Shell Expansion

Commands are executed directly (not via `/bin/sh`):

```go
// GOOD - Direct execution
cmd := exec.Command("npm", "run", "build")

// BAD - Shell injection risk (NOT USED)
cmd := exec.Command("/bin/sh", "-c", "npm run build && rm -rf /")
```

**Config:**
```json
{
  "pre_restart": {
    "command": ["npm", "run", "build"]  // Argv array
  }
}
```

**Rejected:**
```json
{
  "pre_restart": {
    "command": "npm run build && rm -rf /"  // Shell string (not allowed)
  }
}
```

#### 2. No Variable Interpolation

Prevent injection attacks:

```json
{
  "pre_restart": {
    "command": ["echo", "${USER}"]  // Literal string, NOT expanded
  }
}
```

#### 3. Isolated Environment

Pre-restart hooks do NOT inherit Hydra's environment:

```go
cmd := exec.Command(hook.Command[0], hook.Command[1:]...)

// Start with EMPTY environment
cmd.Env = []string{}

// Only add variables from config
for key, value := range hook.Env {
    cmd.Env = append(cmd.Env, key + "=" + value)
}
```

**Example:**
```json
{
  "pre_restart": {
    "command": ["npm", "run", "build"],
    "env": {
      "NODE_ENV": "development",
      "PATH": "/usr/bin:/bin"
    }
  }
}
```

Hook gets ONLY these 2 env vars, not Hydra's full environment.

#### 4. Timeout Enforcement

Prevent infinite hangs:

```go
ctx, cancel := context.WithTimeout(context.Background(), time.Duration(hook.TimeoutMs)*time.Millisecond)
defer cancel()

cmd := exec.CommandContext(ctx, hook.Command[0], hook.Command[1:]...)

if err := cmd.Run(); err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        // Timeout - send SIGKILL
        cmd.Process.Kill()
        return fmt.Errorf("pre-restart hook timed out after %dms", hook.TimeoutMs)
    }
    return err
}
```

**Config:**
```json
{
  "pre_restart": {
    "timeout_ms": 5000  // SIGKILL after 5 seconds
  }
}
```

**Hard Limit:** 30 seconds (config validation rejects higher values)

#### 5. Working Directory Restriction

Hooks run in **server cwd**, NOT Hydra's cwd:

```go
cmd.Dir = filepath.Join(serverConfig.Cwd, hook.Cwd)
```

**Config:**
```json
{
  "cwd": "/Users/dev/projects/my-server",
  "pre_restart": {
    "cwd": "."  // Relative to /Users/dev/projects/my-server
  }
}
```

**Prevents:**
- Hooks running in `/` or other sensitive directories
- Accessing files outside server project

#### 6. Output Truncation

Prevent token bombs:

```go
func CaptureOutput(cmd *exec.Cmd) (string, error) {
    var buf bytes.Buffer
    cmd.Stdout = &buf
    cmd.Stderr = &buf

    err := cmd.Run()

    output := buf.String()
    if len(output) > 10*1024 {  // 10KB max
        output = output[:10*1024] + "\n... [TRUNCATED by Hydra: " + strconv.Itoa(len(output)-10*1024) + " bytes omitted]"
    }

    return output, err
}
```

---

## Process Isolation

### Prevent Zombie Processes

**Problem:** When Hydra kills parent process (e.g., `npm`), child processes (e.g., `node`) may survive.

**Solution:** Process groups + tree kill

#### Unix (Linux/macOS)

```go
cmd := exec.Command(command, args...)

// Create new process group
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,  // Create new process group
}

cmd.Start()

// Later, kill entire process group
pgid, _ := syscall.Getpgid(cmd.Process.Pid)
syscall.Kill(-pgid, syscall.SIGKILL)  // Negative PID = kill group
```

#### Windows

```go
// Use github.com/shirou/gopsutil for cross-platform tree kill

import "github.com/shirou/gopsutil/v3/process"

func KillProcessTree(pid int) error {
    proc, err := process.NewProcess(int32(pid))
    if err != nil {
        return err
    }

    // Get children recursively
    children, _ := proc.Children()

    // Kill children first
    for _, child := range children {
        child.Kill()
    }

    // Kill parent
    return proc.Kill()
}
```

---

## STDIO Injection Prevention

### Threat

Malicious child server sends fake JSON-RPC messages:

```json
{"jsonrpc":"2.0","method":"tools/call","id":999,"params":{"name":"rm -rf /"}}
```

### Mitigation

Hydra validates **direction** of messages:

```go
type Direction int

const (
    DirectionClientToChild Direction = iota
    DirectionChildToClient
)

func ValidateMessage(msg JSONRPCMessage, direction Direction) error {
    switch direction {
    case DirectionChildToClient:
        // Child can only send:
        // - Responses (has "id" + "result" or "error")
        // - Notifications (has "method", no "id")
        if msg.ID != nil && msg.Method != "" {
            return fmt.Errorf("child cannot send requests (has both id and method)")
        }

    case DirectionClientToChild:
        // Client can send:
        // - Requests (has "id" + "method")
        // - Notifications (has "method", no "id")
        // (No validation needed - client is trusted)
    }

    return nil
}
```

**Result:** Child server **cannot** send requests to client (only responses and notifications).

---

## Payload Size Limiting

### Threat

Child server returns massive JSON-RPC result (e.g., 10MB):

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {"data": "... 10MB of text ..."}
}
```

### Mitigation

Hard cap at 50KB (configurable):

```go
func LimitPayloadSize(response JSONRPCResponse, maxKB int) JSONRPCResponse {
    resultJSON, _ := json.Marshal(response.Result)
    sizeKB := len(resultJSON) / 1024

    if sizeKB <= maxKB {
        return response  // OK
    }

    // Exceeded limit - replace result with error
    preview := string(resultJSON[:1024])  // First 1KB
    response.Result = nil
    response.Error = &JSONRPCError{
        Code: -32000,
        Message: fmt.Sprintf(
            "⚠️ Tool output exceeded safety limit (%dKB > %dKB). First 1KB: %s...",
            sizeKB, maxKB, preview,
        ),
    }

    return response
}
```

**Config:**
```json
{
  "behavior": {
    "max_output_size_kb": 50
  }
}
```

---

## Log Rate Limiting

### Threat

Child server spams logs (1000 logs/second):

```python
while True:
    print("DEBUG: " + "X" * 1000)
```

**Cost:** 1000 logs × 1000 chars = 1M chars/s = 250k tokens/s = **$0.75/second** (at Sonnet input rates)

### Mitigation

Token bucket rate limiter:

```go
type RateLimiter struct {
    tokens     int
    maxTokens  int
    refillRate int  // tokens per second
    lastRefill time.Time
    mu         sync.Mutex
}

func (rl *RateLimiter) Allow() bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    elapsed := now.Sub(rl.lastRefill).Seconds()

    // Refill tokens
    rl.tokens = min(rl.maxTokens, rl.tokens + int(elapsed*float64(rl.refillRate)))
    rl.lastRefill = now

    if rl.tokens > 0 {
        rl.tokens--
        return true  // Allow
    }

    return false  // Deny
}
```

**Behavior:**
```
Config: log_rate_limit_per_second=10

T=0s    10 logs → All forwarded (tokens: 10 → 0)
T=0.1s  5 logs  → All denied (tokens: 0)
T=1s    5 logs  → All forwarded (tokens: 10 → 5)
```

**Suppressed logs:**
```
[Hydra] Suppressed 45 rapid-fire logs
```

**Config:**
```json
{
  "behavior": {
    "log_rate_limit_per_second": 10
  }
}
```

---

## Traffic Recorder Privacy

### Default: Disabled

Traffic recorder is **OFF by default** to prevent accidental secret leakage:

```json
{
  "recorder": {
    "enabled": false  // Must explicitly enable
  }
}
```

### When Enabled: Warnings

```bash
hydra run --name my-server --record-traffic
```

**Output:**
```
⚠️  Traffic recording enabled. May capture sensitive data.
    Export will be saved to: /tmp/hydra-traffic-20260121-103045.json
```

### Redaction

Even when enabled, secrets are redacted:

```json
{
  "events": [
    {
      "body": {
        "params": {
          "api_key": "[REDACTED by Hydra]"
        }
      }
    }
  ]
}
```

### Body Redaction by Default

```json
{
  "recorder": {
    "include_request_bodies": false,   // Don't capture
    "include_response_bodies": false   // Don't capture
  }
}
```

**Export shows:**
```json
{
  "events": [
    {
      "method": "tools/call",
      "id": 1,
      "body": "[REDACTED - Set include_request_bodies=true]"
    }
  ]
}
```

---

## Security Best Practices

### For Users

1. **Never commit `~/.hydra/config.json` with secrets**
   - Use `${env:API_KEY}` instead of hardcoded values

2. **Review pre-restart hooks carefully**
   - Understand what commands run
   - Avoid untrusted scripts

3. **Disable traffic recorder in production**
   - Only enable for debugging
   - Delete exports after use

4. **Use .gitignore for Hydra files**
   ```
   .hydra/
   hydra.json
   ```

### For Developers

1. **Never log sensitive data to stderr**
   - Child servers should sanitize logs

2. **Validate tool outputs**
   - Don't return massive payloads

3. **Use environment variables for secrets**
   - Read from `.env` files

---

**End of Security Documentation**
