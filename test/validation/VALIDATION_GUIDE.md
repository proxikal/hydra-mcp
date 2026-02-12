# Hydra Real-World Validation Guide

**Purpose**: Validate Hydra with actual Claude Desktop to prove session resurrection works in real-world usage.

---

## Prerequisites

- Claude Desktop installed
- Hydra built and installed: `/Users/proxikal/dev/projects/hydra/bin/hydra`
- Demo server: `/Users/proxikal/dev/projects/hydra/test/validation/demo_server.py`

---

## Phase 1: Baseline (No Hydra)

**Goal**: Confirm demo server works directly with Claude Desktop.

### Step 1.1: Add Server to Claude Desktop

Edit: `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "preferences": {
    "sidebarMode": "chat"
  },
  "mcpServers": {
    "demo": {
      "command": "python3",
      "args": ["/Users/proxikal/dev/projects/hydra/test/validation/demo_server.py"]
    }
  }
}
```

### Step 1.2: Restart Claude Desktop

Quit and reopen Claude Desktop.

### Step 1.3: Test Tools

In Claude Desktop chat:
```
List available tools
```

**Expected**: Should show `get_current_time` and `crash_test`

```
Use the get_current_time tool
```

**Expected**: Shows current time

### Step 1.4: Test Crash (BASELINE FAILURE)

```
Use the crash_test tool
```

**Expected**:
- Tool responds "Server crashing NOW..."
- Server crashes
- ❌ **Claude Desktop connection dies**
- ❌ **Session lost** (this is the problem Hydra solves!)

---

## Phase 2: With Hydra (The Magic)

**Goal**: Prove Hydra keeps session alive through crashes.

### Step 2.1: Initialize Hydra

```bash
cd /Users/proxikal/dev/projects/hydra
./bin/hydra add demo \
  --command python3 \
  --args /Users/proxikal/dev/projects/hydra/test/validation/demo_server.py \
  --cwd /Users/proxikal/dev/projects/hydra/test/validation
```

This creates `~/.hydra/config.json`:
```json
{
  "servers": {
    "demo": {
      "command": "python3",
      "args": ["/Users/proxikal/dev/projects/hydra/test/validation/demo_server.py"],
      "cwd": "/Users/proxikal/dev/projects/hydra/test/validation",
      "watch": {
        "enabled": false
      },
      "behavior": {
        "max_restarts": 5,
        "restart_window_seconds": 60
      }
    }
  }
}
```

### Step 2.2: Route Through Hydra

```bash
./bin/hydra init --client claude
```

**This will**:
1. Backup Claude config to `claude_desktop_config.json.backup`
2. Modify config to route through Hydra:

```json
{
  "preferences": {
    "sidebarMode": "chat"
  },
  "mcpServers": {
    "demo": {
      "command": "/Users/proxikal/dev/projects/hydra/bin/hydra",
      "args": ["run", "--name", "demo"]
    }
  }
}
```

### Step 2.3: Restart Claude Desktop

Quit and reopen.

### Step 2.4: Test Session Resurrection 🎯

In Claude Desktop:

```
Use the get_current_time tool
```

**Expected**: Works normally

```
Use the crash_test tool
```

**CRITICAL TEST**:
- ✅ Tool responds "Server crashing NOW..."
- ✅ Server crashes (check logs: `hydra logs demo`)
- ✅ **Session STAYS ALIVE** (this is Hydra's magic!)
- ✅ You can continue chatting without reconnecting

```
Use the get_current_time tool again
```

**Expected**: ✅ Works! Session survived the crash!

---

## Phase 3: Advanced Validation

### Test 3.1: Multiple Crashes

Run `crash_test` 3 times in a row.

**Expected**:
- All 3 crashes handled
- Session stays alive
- Check restarts: `hydra status demo`
- Should show `restarts: 3`

### Test 3.2: Crash Loop Detection

Run `crash_test` 6+ times rapidly.

**Expected**:
- After 5 restarts (max_restarts), Hydra enters FAILED state
- Error message shown to Claude
- Recovery: `hydra recover demo`

### Test 3.3: Hot Reload (if watcher enabled)

1. Enable watcher in `~/.hydra/config.json`:
```json
{
  "servers": {
    "demo": {
      ...
      "watch": {
        "enabled": true,
        "paths": ["/Users/proxikal/dev/projects/hydra/test/validation"],
        "extensions": [".py"]
      }
    }
  }
}
```

2. Edit `demo_server.py` (add a comment)
3. Save file

**Expected**:
- Hydra detects change
- Restarts server automatically
- Session stays alive in Claude
- Check logs: `hydra logs demo` shows "file change detected"

### Test 3.4: Injectable Tools

In Claude Desktop:
```
Use the hydra_status tool
```

**Expected**: Shows Hydra supervisor status

```
Use the hydra_logs tool
```

**Expected**: Shows last 20 lines of server logs

```
Use the hydra_restart tool
```

**Expected**: Manually triggers restart, session survives

---

## Validation Checklist

Mark each test:

- [ ] Phase 1: Baseline crash kills session (expected failure)
- [ ] Phase 2.4: Crash with Hydra keeps session alive ✅
- [ ] Phase 3.1: Multiple crashes handled
- [ ] Phase 3.2: Crash loop detection works
- [ ] Phase 3.3: Hot reload works (optional)
- [ ] Phase 3.4: Injectable tools accessible

---

## Success Criteria

**Hydra is validated if**:
1. ✅ Session survives at least one server crash
2. ✅ Claude Desktop can continue using tools after crash
3. ✅ Injectable tools (`hydra_status`, `hydra_logs`) work
4. ✅ Crash loop detection prevents infinite restarts

---

## Rollback

To remove Hydra and restore original config:

```bash
./bin/hydra uninit --client claude
```

This restores from backup and removes Hydra routing.

---

## Troubleshooting

**Server won't start:**
```bash
# Check Hydra logs
hydra logs demo --follow

# Check if demo server works standalone
python3 /Users/proxikal/dev/projects/hydra/test/validation/demo_server.py
# Then type: {"jsonrpc":"2.0","id":1,"method":"initialize"}
```

**Session still dies on crash:**
- Verify Hydra is in the path: `which hydra`
- Check Claude config shows hydra command, not python3 directly
- Restart Claude Desktop after config changes

**Tools not appearing:**
- Check server initialized: `hydra status demo`
- Check Claude Desktop MCP sidebar shows "demo" server
- Restart Claude Desktop

---

## Expected Logs

When crash_test is called, Hydra logs should show:

```
{"level":"info","message":"request forwarded to child"}
{"level":"error","error":"child eof","message":"child process exited"}
{"level":"info","message":"restarting child process"}
{"level":"info","message":"child process restarted"}
{"level":"info","message":"state replayed"}
```

Claude Desktop should NOT show disconnection - session continues seamlessly.

---

## Metrics

After validation, check metrics:

```bash
hydra inspect demo
```

Should show:
- Restarts: >0 (from crash tests)
- Requests: multiple
- Health: >95%
- Uptime: should be continuous (Hydra uptime, not child uptime)

---

**Next Steps**: After validation, document results and create demo video.
