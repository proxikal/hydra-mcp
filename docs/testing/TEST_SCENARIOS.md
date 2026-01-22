# Hydra Test Scenarios

Complete test matrix covering all edge cases and failure modes.

---

## Unit Test Scenarios

### Sanitizer

| Test Case | Input | Expected Output |
|-----------|-------|-----------------|
| `test_valid_jsonrpc_request` | `{"jsonrpc":"2.0","method":"initialize","id":1}` | ChunkJSONRPC |
| `test_valid_jsonrpc_response` | `{"jsonrpc":"2.0","id":1,"result":{}}` | ChunkJSONRPC |
| `test_valid_jsonrpc_notification` | `{"jsonrpc":"2.0","method":"notify"}` | ChunkJSONRPC |
| `test_pollution_log` | `DEBUG: Starting server...` | ChunkPollution |
| `test_pollution_stacktrace` | `Traceback (most recent call last):...` | ChunkPollution |
| `test_empty_line` | `` | ChunkEmpty |
| `test_whitespace_only` | `   \n  ` | ChunkEmpty |
| `test_valid_json_not_jsonrpc` | `{"foo":"bar"}` | ChunkPollution |
| `test_invalid_utf8` | `\xFF\xFE invalid` | ChunkPollution (after UTF-8 validation) |

### StateStore

| Test Case | Actions | Expected Behavior |
|-----------|---------|-------------------|
| `test_initialize_replay` | SetInitialize → GetInitialize | Returns cached params |
| `test_subscription_add` | AddSubscription(uri1) | Subscription cached |
| `test_subscription_multiple` | Add(uri1), Add(uri2) → GetSubscriptions | Returns [uri1, uri2] |
| `test_subscription_remove` | Add(uri1), Remove(uri1) → GetSubscriptions | Returns [] |
| `test_clear_all` | SetInitialize, AddSubscription → Clear → Get* | Returns nil/[] |

### Supervisor (Mocked)

| Test Case | Setup | Action | Expected |
|-----------|-------|--------|----------|
| `test_start_success` | Mock returns nil | Start() | State = STARTING |
| `test_start_failure` | Mock returns error | Start() | Returns error |
| `test_restart_increments_counter` | - | Restart() × 3 | Counter = 3 |
| `test_crash_loop_detection` | max=3 | Restart() × 4 | InCrashLoop() = true |
| `test_crash_loop_window` | max=3, window=60s | 4 restarts in 30s | InCrashLoop() = true |
| `test_crash_loop_window_expired` | max=3, window=2s | 3 restarts, wait 3s, 1 restart | InCrashLoop() = false |

### Debouncer

| Test Case | Events | Expected Restarts |
|-----------|--------|-------------------|
| `test_single_change` | Change at T=0 | 1 restart at T=500ms |
| `test_rapid_changes` | Changes at T=0, T=100, T=300 | 1 restart at T=800ms |
| `test_batch_window_expires` | Changes spread over 3s (window=2s) | 2 restarts |
| `test_cooldown_prevents_restart` | Change at T=0, restart at T=500, change at T=600 (cooldown=5s) | 1 restart (2nd ignored) |

---

## Integration Test Scenarios

### 1. Crash Loop Detection

**Fixture:** `crash_server.py` (always exits with code 1)

**Config:**
```json
{
  "command": "python",
  "args": ["crash_server.py"],
  "behavior": {
    "max_restarts": 3,
    "restart_window_seconds": 60
  }
}
```

**Test:**
```
1. Start Hydra
2. Wait 10 seconds
3. Assert state = FAILED
4. Assert restarts > 3
5. Assert error contains "exit status 1"
```

---

### 2. File Watch Batching (Git Checkout Simulation)

**Fixture:** Temp directory with 50 Python files

**Test:**
```
1. Start Hydra watching temp dir
2. Wait for RUNNING state
3. Modify all 50 files within 500ms
4. Wait 2 seconds
5. Assert restart count = 1 (not 50)
```

---

### 3. Subscription Resurrection

**Fixture:** `echo_server.py` with subscription support

**Test:**
```
1. Start Hydra + echo_server
2. Client: Send resources/subscribe(uri="file:///test.txt")
3. Server: Returns subscription_id=1
4. Kill child process (simulate crash)
5. Wait for RESTARTING → RUNNING
6. Assert child received resources/subscribe(uri="file:///test.txt") again
7. Assert new subscription_id ≠ 1
```

---

### 4. Request Queueing During Restart

**Fixture:** `echo_server.py`

**Test:**
```
1. Start Hydra
2. Wait for RUNNING
3. Trigger restart (file change)
4. While RESTARTING:
   - Send tools/call(id=1)
   - Send tools/call(id=2)
   - Send tools/call(id=3)
5. Wait for RUNNING
6. Assert all 3 responses received in order
```

---

### 5. Queue Overflow

**Fixture:** `slow_server.py` (30s startup)

**Config:**
```json
{
  "behavior": {
    "restart_queue_max_messages": 10
  }
}
```

**Test:**
```
1. Start Hydra + slow_server
2. Trigger restart
3. While RESTARTING:
   - Send 15 requests (id=1..15)
4. Assert first 10 are queued
5. Assert requests 11-15 get error: "Server restarting, queue full"
6. Wait for RUNNING
7. Assert first 10 requests get responses
```

---

### 6. Slow Startup with Queue

**Fixture:** `slow_server.py` (30s startup)

**Test:**
```
1. Start Hydra
2. Immediately send tools/call(id=1)
3. Assert request is queued (STARTING state)
4. Wait 30 seconds for initialize
5. Assert state = RUNNING
6. Assert request id=1 gets response
```

---

### 7. Pre-Restart Hook Success

**Fixture:** `echo_server.py`

**Config:**
```json
{
  "pre_restart": {
    "enabled": true,
    "command": ["echo", "Building..."],
    "timeout_ms": 5000
  }
}
```

**Test:**
```
1. Start Hydra
2. Trigger restart
3. Assert hook runs successfully
4. Assert restart proceeds
5. Assert state = RUNNING
```

---

### 8. Pre-Restart Hook Timeout

**Config:**
```json
{
  "pre_restart": {
    "enabled": true,
    "command": ["sleep", "60"],
    "timeout_ms": 1000
  }
}
```

**Test:**
```
1. Start Hydra
2. Trigger restart
3. Assert hook is killed after 1000ms
4. Assert restart proceeds (on_error = "warn_and_continue")
```

---

### 9. Pre-Restart Hook Failure (Abort)

**Config:**
```json
{
  "pre_restart": {
    "enabled": true,
    "command": ["false"],  // Always fails
    "on_error": "abort"
  }
}
```

**Test:**
```
1. Start Hydra (RUNNING)
2. Trigger restart
3. Assert hook fails
4. Assert restart is aborted
5. Assert state = RUNNING (old process still alive)
```

---

### 10. Chatty Server (Log Rate Limiting)

**Fixture:** `chatty_server.py` (100 logs/second)

**Config:**
```json
{
  "behavior": {
    "log_rate_limit_per_second": 10
  }
}
```

**Test:**
```
1. Start Hydra + chatty_server
2. Let run for 5 seconds
3. Count logs forwarded to client
4. Assert ~50 logs forwarded (10/s × 5s)
5. Assert "Suppressed N logs" messages appear
```

---

### 11. Large Payload Truncation

**Fixture:** Custom server that returns 100KB result

**Config:**
```json
{
  "behavior": {
    "max_output_size_kb": 50
  }
}
```

**Test:**
```
1. Start Hydra
2. Call tool that returns 100KB
3. Assert response is error
4. Assert error contains "exceeded safety limit (100KB > 50KB)"
5. Assert error includes first 1KB preview
```

---

### 12. Secret Redaction

**Fixture:** Server that logs `API_KEY=sk-abc123`

**Config:**
```json
{
  "security": {
    "redact_patterns": ["sk-[A-Za-z0-9]+"]
  }
}
```

**Test:**
```
1. Start Hydra
2. Server logs "API_KEY=sk-abc123"
3. Call hydra_logs tool
4. Assert logs contain "API_KEY=[REDACTED by Hydra]"
```

---

### 13. Protocol Auto-Detection (NDJSON)

**Fixture:** `echo_server.py` (NDJSON format)

**Test:**
```
1. Start Hydra (protocol=auto)
2. Wait for first message from server
3. Assert detected protocol = NDJSON
4. Assert communication works
```

---

### 14. Protocol Auto-Detection (LSP)

**Fixture:** LSP-style server (Content-Length headers)

**Test:**
```
1. Start Hydra (protocol=auto)
2. Wait for first message
3. Assert detected protocol = LSP
4. Assert communication works
```

---

### 15. Tool Namespace Collision

**Fixture:** Server that exposes `hydra_restart` tool

**Config:**
```json
{
  "injectable_tools": {
    "on_collision": "error"
  }
}
```

**Test:**
```
1. Start Hydra
2. Assert Hydra refuses to start
3. Assert error contains "Namespace collision: hydra_restart"
```

---

## Chaos Test Scenarios

### 16. Intermittent STDIO Corruption

**Fixture:** Server that randomly prints logs between JSON-RPC messages

**Test:**
```
1. Start Hydra
2. Send 100 requests
3. Server injects random logs 30% of the time
4. Assert all 100 responses received correctly
5. Assert no JSON parse errors
```

---

### 17. SIGTERM Hang (Tree Kill)

**Fixture:** Server that ignores SIGTERM

**Test:**
```
1. Start Hydra
2. Trigger restart
3. Server ignores SIGTERM
4. Assert Hydra sends SIGKILL after graceful_shutdown_ms
5. Assert process is killed
6. Assert restart proceeds
```

---

### 18. Memory Leak Detection

**Fixture:** `echo_server.py`

**Test:**
```
1. Start Hydra
2. Measure initial RSS memory
3. Trigger 1000 restarts (1 per second)
4. Measure final RSS memory
5. Assert memory increase < 100MB
```

---

### 19. Real Claude Desktop Integration

**Prerequisites:** Claude Desktop installed

**Test:**
```
1. Run `hydra init --client claude --dry-run`
2. Capture proposed changes
3. Apply changes
4. Start Claude Desktop
5. Connect to server via Hydra
6. Trigger 10 restarts
7. Assert Claude session stays alive
8. Assert tools still work after restarts
```

---

## Benchmark Scenarios

### 20. Proxy Overhead

**Measurement:** Latency added by Hydra

**Test:**
```
For i = 1 to 1000:
  Send request to child via Hydra
  Measure round-trip time
  Record latency

Calculate p50, p99
Assert p50 < 50ms
Assert p99 < 200ms
```

---

### 21. Restart Speed

**Measurement:** Time from RESTARTING → RUNNING

**Test:**
```
For i = 1 to 100:
  Trigger restart
  Measure time from state=RESTARTING to state=RUNNING
  Record duration

Calculate p50, p99
Assert p50 < 500ms
Assert p99 < 2s
```

---

### 22. File Watch Latency

**Measurement:** Time from file change to restart trigger

**Test:**
```
For i = 1 to 100:
  Modify watched file
  Measure time until restart begins
  Record latency

Calculate p50, p99
Assert p50 < 100ms
Assert p99 < 500ms
```

---

## Edge Case Scenarios

### 23. Config Hot-Reload

**Test:**
```
1. Start Hydra with config A
2. Modify ~/.hydra/config.json (change debounce_ms)
3. Send SIGHUP to Hydra
4. Assert config reloaded
5. Assert new debounce value applied
```

---

### 24. Multiple Servers (Isolation)

**Test:**
```
1. Start Hydra instance #1 (server A)
2. Start Hydra instance #2 (server B)
3. Crash server A
4. Assert server B unaffected
5. Assert instance #1 restarts only server A
```

---

### 25. Concurrent Requests During Restart

**Test:**
```
1. Start Hydra
2. Trigger restart
3. While RESTARTING, send 10 concurrent requests
4. Assert all 10 are queued
5. Assert all 10 get responses after RUNNING
6. Assert responses are correct (no mixing)
```

---

### 26. Subscription During Restart

**Test:**
```
1. Start Hydra (RUNNING)
2. Subscribe to resource A
3. Trigger restart
4. While RESTARTING, subscribe to resource B
5. Assert resource B subscription queued
6. After RUNNING:
   - Assert resource A re-subscribed
   - Assert resource B subscribed
```

---

**End of Test Scenarios**
