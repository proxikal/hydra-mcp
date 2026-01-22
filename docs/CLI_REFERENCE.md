# Hydra CLI Reference

Complete reference for all Hydra CLI commands.

---

## Core Commands

### `hydra run`

Start Hydra supervisor for a server.

**Usage:**
```bash
hydra run --name <server-name> [flags]
```

**Flags:**
- `--name` (required) - Server name from registry
- `--debug` - Enable debug logging to stderr
- `--record-traffic` - Enable traffic recorder

**Examples:**
```bash
# Start server
hydra run --name my-python-server

# Start with debug logging
hydra run --name my-python-server --debug

# Start with traffic recording
hydra run --name my-python-server --record-traffic
```

**Behavior:**
1. Load `~/.hydra/config.json`
2. Find server entry by name
3. Load `./hydra.json` if present
4. Merge configs
5. Validate config
6. Spawn child process
7. Enter STARTING state
8. Wait for initialize response
9. Enter RUNNING state

**Exit Codes:**
- `0` - Clean shutdown (SIGINT/SIGTERM)
- `1` - Config error / validation failed
- `2` - Child process in crash loop (if on_crash_loop="exit")

---

### `hydra init`

Bootstrap Hydra for an AI client.

**Usage:**
```bash
hydra init --client <client-type> [flags]
```

**Flags:**
- `--client` (required) - Client type: `claude`, `cline`, `cursor`, `custom`
- `--config-path` - Path to custom client config (required if `--client custom`)
- `--dry-run` - Show changes without applying

**Examples:**
```bash
# Initialize for Claude Desktop
hydra init --client claude

# Dry run (preview changes)
hydra init --client claude --dry-run

# Custom AI client
hydra init --client custom --config-path ~/my-ai-client.json
```

**Behavior:**
1. Locate client config file
2. Parse existing MCP server entries
3. Prompt user to select servers for Hydra supervision
4. Create `~/.hydra/config.json` if doesn't exist
5. Add selected servers to registry
6. Backup client config to `{config}.backup`
7. Modify client config to use Hydra:
   - Before: `"command": "python", "args": ["server.py"]`
   - After: `"command": "hydra", "args": ["run", "--name", "my-server"]`
8. Display summary of changes

**Supported Clients:**
- `claude` → `~/.config/claude/claude_desktop_config.json`
- `cline` → VS Code Cline extension config
- `cursor` → Cursor editor config
- `custom` → User-specified path

**Safety:**
- Always creates backup before modifying
- Refuses to run if backup already exists (prevents double-init)
- Validates JSON before writing

---

### `hydra uninit`

Rollback `hydra init` (restore from backup).

**Usage:**
```bash
hydra uninit --client <client-type>
```

**Examples:**
```bash
hydra uninit --client claude
```

**Behavior:**
1. Check if backup exists (`{config}.backup`)
2. Restore backup to original location
3. Remove backup file
4. Display summary

**Note:** Does NOT remove servers from `~/.hydra/config.json`

---

## Server Management

### `hydra add`

Add server to registry.

**Usage:**
```bash
hydra add <server-name> [flags]
```

**Flags:**
- `--command` (required) - Executable command
- `--args` - Command arguments (repeatable)
- `--cwd` (required) - Working directory
- `--watch-path` - Watch directory (repeatable)
- `--watch-ext` - Watch file extension (repeatable)
- `--env` - Environment variable KEY=VALUE (repeatable)

**Examples:**
```bash
# Add Python server
hydra add my-python-server \
  --command python \
  --args server.py \
  --cwd ~/projects/my-server \
  --watch-path ./src \
  --watch-ext .py

# Add Node server with multiple watch paths
hydra add my-node-server \
  --command node \
  --args dist/index.js \
  --cwd ~/projects/my-node-server \
  --watch-path ./src \
  --watch-path ./lib \
  --watch-ext .ts \
  --watch-ext .js \
  --env NODE_ENV=development
```

**Behavior:**
1. Validate command exists and is executable
2. Validate cwd exists
3. Create server entry in `~/.hydra/config.json`
4. Apply global defaults
5. Display summary

---

### `hydra list`

List all servers in registry.

**Usage:**
```bash
hydra list
```

**Output:**
```
my-python-server   RUNNING    PID: 12345   Uptime: 2h 15m   Restarts: 2/10
my-node-server     RESTARTING PID: -       Restarts: 5/10
old-server         STOPPED    PID: -
```

**Columns:**
- Server name
- State (STOPPED, STARTING, RUNNING, RESTARTING, FAILED)
- PID (process ID, or "-" if not running)
- Uptime (if RUNNING)
- Restarts (current/max in window)

---

### `hydra remove`

Remove server from registry.

**Usage:**
```bash
hydra remove <server-name>
```

**Examples:**
```bash
hydra remove my-old-server
```

**Behavior:**
1. Remove server entry from `~/.hydra/config.json`
2. Display confirmation

**Note:** Does NOT modify AI client config. User must:
- Manually update AI client config, OR
- Run `hydra init` again to sync

---

### `hydra edit`

Edit server config in $EDITOR.

**Usage:**
```bash
hydra edit <server-name>
```

**Examples:**
```bash
hydra edit my-python-server
```

**Behavior:**
1. Open `~/.hydra/config.json` in `$EDITOR`
2. Position cursor at server entry (if editor supports)
3. Wait for editor to close
4. Validate config
5. Display validation result

**Editor Detection:**
1. `$EDITOR` environment variable
2. `vim` (fallback)

---

### `hydra validate`

Validate server configuration.

**Usage:**
```bash
hydra validate <server-name>
```

**Examples:**
```bash
hydra validate my-python-server
```

**Checks:**
- Command exists and is executable
- CWD exists
- Watch paths exist
- Environment variables resolve (${env:...})
- Pre-restart command is valid (if enabled)
- Config schema is valid (JSON schema validation)

**Output:**
```
✓ Command exists: /usr/bin/python
✓ CWD exists: /Users/dev/projects/my-server
✓ Watch paths exist
✓ Environment variables valid
⚠ Warning: max_restarts is very low (3)
✓ Config is valid
```

**Exit Codes:**
- `0` - Valid
- `1` - Validation failed

---

## Debugging Commands

### `hydra logs`

View child server logs.

**Usage:**
```bash
hydra logs <server-name> [flags]
```

**Flags:**
- `--follow` - Tail logs in real-time (like `tail -f`)
- `--stderr` - Show stderr only (default: both stdout and stderr)
- `--lines N` - Number of lines to show (default: 50)

**Examples:**
```bash
# Show last 50 lines
hydra logs my-python-server

# Follow logs
hydra logs my-python-server --follow

# Show last 100 stderr lines
hydra logs my-python-server --stderr --lines 100
```

**Output:**
```
[2026-01-21 10:30:45] INFO: Server starting...
[2026-01-21 10:30:46] DEBUG: Loading config...
[2026-01-21 10:30:47] ERROR: Connection failed
```

---

### `hydra status`

Show server status.

**Usage:**
```bash
hydra status <server-name>
```

**Examples:**
```bash
hydra status my-python-server
```

**Output (JSON):**
```json
{
  "state": "RUNNING",
  "server_name": "my-python-server",
  "pid": 12345,
  "uptime_seconds": 7200,
  "restarts_in_window": 2,
  "max_restarts": 10,
  "restart_window_seconds": 60,
  "last_restart_reason": "file_change",
  "last_error": null,
  "queue_size": 0,
  "can_recover": false
}
```

**Fields:**
- `state` - Current state (STOPPED, STARTING, RUNNING, RESTARTING, FAILED)
- `server_name` - Server name
- `pid` - Process ID (0 if not running)
- `uptime_seconds` - Uptime in seconds (0 if not running)
- `restarts_in_window` - Restart count in current window
- `max_restarts` - Max restarts before crash loop
- `restart_window_seconds` - Window size in seconds
- `last_restart_reason` - Reason for last restart ("file_change", "crash", "manual")
- `last_error` - Last error message (null if no error)
- `queue_size` - Number of queued requests (RESTARTING state)
- `can_recover` - Whether server can be recovered from FAILED state

---

### `hydra restart`

Manually restart server.

**Usage:**
```bash
hydra restart <server-name> [flags]
```

**Flags:**
- `--reason` - Optional reason for restart (logged for debugging)

**Examples:**
```bash
# Restart server
hydra restart my-python-server

# Restart with reason
hydra restart my-python-server --reason "Testing new config"
```

**Behavior:**
1. If state = RUNNING:
   - Trigger restart (same as file change)
   - Record reason
2. If state = STOPPED:
   - Error: "Server not running"
3. If state = FAILED:
   - Error: "Server in FAILED state, use 'hydra recover'"

---

### `hydra recover`

Recover from FAILED state.

**Usage:**
```bash
hydra recover <server-name>
```

**Examples:**
```bash
hydra recover my-python-server
```

**Behavior:**
1. Check state = FAILED
2. Reset restart counter
3. Clear StateStore
4. Transition FAILED → STOPPED
5. Attempt restart (STOPPED → STARTING)
6. Display result

**Output:**
```
✓ Reset restart counter
✓ Cleared state
✓ Attempting restart...
✓ Server recovered (state: RUNNING)
```

---

### `hydra export-traffic`

Export traffic recorder buffer.

**Usage:**
```bash
hydra export-traffic <server-name> [flags]
```

**Flags:**
- `--output` - Output file path (default: stdout)

**Examples:**
```bash
# Export to stdout
hydra export-traffic my-python-server

# Export to file
hydra export-traffic my-python-server --output debug.json
```

**Output Format:**
```json
{
  "hydra_version": "1.0.0",
  "server_name": "my-python-server",
  "recorded_at": "2026-01-21T10:30:45Z",
  "warning": "This file may contain sensitive data. Do not share publicly.",
  "events": [
    {
      "timestamp": "2026-01-21T10:30:46Z",
      "direction": "client→hydra",
      "method": "initialize",
      "id": 1,
      "body": "[REDACTED - Set include_request_bodies=true]"
    }
  ]
}
```

**Note:** Requires `recorder.enabled: true` in config

---

## Global Flags

All commands support these global flags:

- `--help, -h` - Show help
- `--version, -v` - Show version
- `--config` - Path to global config (default: `~/.hydra/config.json`)

**Examples:**
```bash
hydra --help
hydra --version
hydra run --name my-server --config ~/custom-config.json
```

---

## Environment Variables

- `HYDRA_CONFIG` - Path to global config (overrides `~/.hydra/config.json`)
- `EDITOR` - Editor for `hydra edit` command
- `NO_COLOR` - Disable colored output

**Examples:**
```bash
export HYDRA_CONFIG=~/custom-config.json
hydra list

export EDITOR=nano
hydra edit my-server

export NO_COLOR=1
hydra list
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (config invalid, validation failed, etc.) |
| 2 | Crash loop (if `on_crash_loop: "exit"`) |
| 130 | Interrupted (SIGINT / Ctrl+C) |

---

**End of CLI Reference**
