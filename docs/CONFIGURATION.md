# Hydra Configuration Reference

Complete configuration schema and examples for Hydra.

---

## Global Registry (`~/.hydra/config.json`)

**Location:** `~/.hydra/config.json`

**Purpose:** Single source of truth for all MCP servers supervised by Hydra.

### Complete Schema

```json
{
  "version": "1.0",
  "defaults": {
    "debounce_ms": 500,
    "graceful_shutdown_ms": 2000,
    "max_output_size_kb": 50,
    "log_rate_limit_per_second": 10,
    "restart_queue_max_messages": 100,
    "restart_queue_ttl_seconds": 30,
    "restart_timeout_seconds": 10
  },
  "servers": {
    "my-python-server": {
      "command": "python",
      "args": ["server.py"],
      "cwd": "/Users/dev/projects/mcp-server-python",
      "env_file": ".env",
      "environment": {
        "DEBUG": "true",
        "API_KEY": "${env:API_KEY}",
        "PATH": "${env:PATH}"
      },
      "watch": {
        "enabled": true,
        "paths": ["./src", "./lib"],
        "extensions": [".py", ".json"],
        "ignore_patterns": ["**/__pycache__", "**/*.pyc", "**/*.log"],
        "use_gitignore": true
      },
      "behavior": {
        "debounce_ms": 500,
        "batch_window_ms": 2000,
        "cooldown_after_restart_ms": 5000,
        "graceful_shutdown_ms": 2000,
        "max_restarts": 10,
        "restart_window_seconds": 60,
        "on_crash_loop": "pause"
      },
      "pre_restart": {
        "enabled": false,
        "command": ["find", ".", "-name", "*.pyc", "-delete"],
        "timeout_ms": 5000,
        "on_error": "abort",
        "cwd": ".",
        "env": {}
      },
      "transport": {
        "protocol": "auto",
        "auto_detect_timeout_ms": 1000
      },
      "injectable_tools": {
        "enabled": true,
        "tools": ["hydra_restart", "hydra_logs", "hydra_status"],
        "on_collision": "error"
      },
      "recorder": {
        "enabled": false,
        "buffer_size": 50,
        "include_request_bodies": false,
        "include_response_bodies": false,
        "export_on_crash": true,
        "export_path": "/tmp/hydra-traffic-{timestamp}.json"
      },
      "security": {
        "redact_patterns": [
          "sk-[A-Za-z0-9]{32,}",
          "API[_-]?KEY",
          "password",
          "Bearer [A-Za-z0-9._-]+"
        ],
        "redact_replacement": "[REDACTED by Hydra]"
      }
    }
  }
}
```

---

## Field Reference

### Top Level

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | Config schema version (e.g., "1.0") |
| `defaults` | object | No | Global defaults applied to all servers |
| `servers` | object | Yes | Map of server name → server config |

### defaults

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `debounce_ms` | int | 500 | Wait time after last file change before restart |
| `graceful_shutdown_ms` | int | 2000 | Wait time for SIGTERM before SIGKILL |
| `max_output_size_kb` | int | 50 | Max JSON-RPC result size before truncation |
| `log_rate_limit_per_second` | int | 10 | Max logs forwarded per second |
| `restart_queue_max_messages` | int | 100 | Max requests queued during restart |
| `restart_queue_ttl_seconds` | int | 30 | Max age of queued request |
| `restart_timeout_seconds` | int | 10 | Max time for restart before FAILED |

### servers.{name}

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | Yes | Executable command (e.g., "python", "node") |
| `args` | string[] | Yes | Command arguments |
| `cwd` | string | Yes | Working directory (absolute path) |
| `env_file` | string | No | Path to .env file (relative to cwd) |
| `environment` | object | No | Environment variables (key-value pairs) |
| `watch` | object | No | File watching configuration |
| `behavior` | object | No | Restart behavior |
| `pre_restart` | object | No | Pre-restart hook configuration |
| `transport` | object | No | Protocol configuration |
| `injectable_tools` | object | No | Hydra tool injection |
| `recorder` | object | No | Traffic recorder |
| `security` | object | No | Security settings |

### watch

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | true | Enable file watching |
| `paths` | string[] | [] | Directories to watch (relative to cwd) |
| `extensions` | string[] | [] | File extensions to watch (e.g., [".py", ".js"]) |
| `ignore_patterns` | string[] | [] | Glob patterns to ignore |
| `use_gitignore` | bool | false | Parse .gitignore and ignore those files |

### behavior

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `debounce_ms` | int | 500 | Wait after last file change |
| `batch_window_ms` | int | 2000 | Max time to batch changes |
| `cooldown_after_restart_ms` | int | 5000 | Ignore changes for X ms after restart |
| `graceful_shutdown_ms` | int | 2000 | SIGTERM → SIGKILL timeout |
| `max_restarts` | int | 10 | Max restarts in window before FAILED |
| `restart_window_seconds` | int | 60 | Time window for counting restarts |
| `on_crash_loop` | string | "pause" | Action on crash loop: "pause", "exit", "notify" |

### pre_restart

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable pre-restart hook |
| `command` | string[] | [] | Command argv (NOT shell string) |
| `timeout_ms` | int | 5000 | Command timeout |
| `on_error` | string | "abort" | Action on error: "abort", "warn_and_continue", "ignore" |
| `cwd` | string | "." | Working directory (relative to server cwd) |
| `env` | object | {} | Environment variables for hook |

### transport

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `protocol` | string | "auto" | Protocol: "auto", "ndjson", "lsp" |
| `auto_detect_timeout_ms` | int | 1000 | Timeout for auto-detection |

### injectable_tools

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | true | Enable tool injection |
| `tools` | string[] | ["hydra_restart", "hydra_logs", "hydra_status"] | Tools to inject |
| `on_collision` | string | "error" | Action on namespace collision: "error", "warn", "disable_hydra_tool" |

### recorder

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | false | Enable traffic recording |
| `buffer_size` | int | 50 | Circular buffer size (number of messages) |
| `include_request_bodies` | bool | false | Include full request bodies in export |
| `include_response_bodies` | bool | false | Include full response bodies in export |
| `export_on_crash` | bool | true | Auto-export on crash |
| `export_path` | string | "/tmp/hydra-traffic-{timestamp}.json" | Export file path |

### security

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `redact_patterns` | string[] | ["sk-[A-Za-z0-9]{32,}", ...] | Regex patterns for secret redaction |
| `redact_replacement` | string | "[REDACTED by Hydra]" | Replacement text for redacted secrets |

---

## Local Override (`./hydra.json`)

**Location:** `$CWD/hydra.json` (where server code lives)

**Purpose:** Per-project development overrides.

### Schema

```json
{
  "name": "my-python-server",
  "watch": {
    "paths": ["./src", "./lib", "./tests"]
  },
  "behavior": {
    "debounce_ms": 200
  },
  "recorder": {
    "enabled": true
  }
}
```

### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Must match server name in global registry |
| All other fields | - | No | Same as global registry server config |

**Merge Priority:** `defaults < registry entry < local override`

---

## Environment Variable Substitution

### Syntax

```
${env:VARIABLE_NAME}
${env:VARIABLE_NAME:default_value}
```

### Rules

1. **Single-pass substitution** (no recursion)
2. **Only `${env:...}` supported** (no shell expansion like `~`, `*`, `$HOME`)
3. **Missing variables without defaults** → fail-fast at config load
4. **Escape literal `$`** with `\\${...}`
5. **Substitution happens after .env file** is loaded

### Load Order

```
1. Load .env file (if env_file specified)
2. Merge config.environment with .env (config wins)
3. Substitute ${env:...} from Hydra's process environment
4. Pass final environment to child process
```

### Examples

```json
{
  "environment": {
    "LITERAL": "hardcoded-value",
    "FROM_HOST": "${env:HOME}",
    "WITH_DEFAULT": "${env:API_KEY:sk-default}",
    "ESCAPED": "\\${not-a-variable}",
    "PATH_APPEND": "${env:PATH}:/custom/bin"
  }
}
```

**Result:**

```
LITERAL=hardcoded-value
FROM_HOST=/Users/dev
WITH_DEFAULT=sk-abc123 (or sk-default if API_KEY not set)
ESCAPED=${not-a-variable}
PATH_APPEND=/usr/bin:/bin:/usr/sbin:/sbin:/custom/bin
```

---

## JSON Schema (for Validation)

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://hydra.mcp.dev/schema.json",
  "title": "Hydra Configuration",
  "type": "object",
  "required": ["version", "servers"],
  "properties": {
    "version": {
      "type": "string",
      "pattern": "^[0-9]+\\.[0-9]+$"
    },
    "defaults": {
      "type": "object",
      "properties": {
        "debounce_ms": { "type": "integer", "minimum": 0, "maximum": 10000 },
        "graceful_shutdown_ms": { "type": "integer", "minimum": 0, "maximum": 30000 },
        "max_output_size_kb": { "type": "integer", "minimum": 1, "maximum": 1024 },
        "log_rate_limit_per_second": { "type": "integer", "minimum": 1, "maximum": 1000 }
      }
    },
    "servers": {
      "type": "object",
      "additionalProperties": {
        "$ref": "#/definitions/server"
      }
    }
  },
  "definitions": {
    "server": {
      "type": "object",
      "required": ["command", "args", "cwd"],
      "properties": {
        "command": { "type": "string", "minLength": 1 },
        "args": { "type": "array", "items": { "type": "string" } },
        "cwd": { "type": "string", "minLength": 1 },
        "env_file": { "type": "string" },
        "environment": { "type": "object", "additionalProperties": { "type": "string" } },
        "watch": { "$ref": "#/definitions/watch" },
        "behavior": { "$ref": "#/definitions/behavior" },
        "pre_restart": { "$ref": "#/definitions/pre_restart" },
        "transport": { "$ref": "#/definitions/transport" },
        "injectable_tools": { "$ref": "#/definitions/injectable_tools" },
        "recorder": { "$ref": "#/definitions/recorder" },
        "security": { "$ref": "#/definitions/security" }
      }
    },
    "watch": {
      "type": "object",
      "properties": {
        "enabled": { "type": "boolean" },
        "paths": { "type": "array", "items": { "type": "string" } },
        "extensions": { "type": "array", "items": { "type": "string", "pattern": "^\\..+" } },
        "ignore_patterns": { "type": "array", "items": { "type": "string" } },
        "use_gitignore": { "type": "boolean" }
      }
    },
    "behavior": {
      "type": "object",
      "properties": {
        "debounce_ms": { "type": "integer", "minimum": 0, "maximum": 10000 },
        "batch_window_ms": { "type": "integer", "minimum": 0, "maximum": 30000 },
        "cooldown_after_restart_ms": { "type": "integer", "minimum": 0, "maximum": 60000 },
        "graceful_shutdown_ms": { "type": "integer", "minimum": 0, "maximum": 30000 },
        "max_restarts": { "type": "integer", "minimum": 1, "maximum": 100 },
        "restart_window_seconds": { "type": "integer", "minimum": 1, "maximum": 600 },
        "on_crash_loop": { "enum": ["pause", "exit", "notify"] }
      }
    },
    "pre_restart": {
      "type": "object",
      "properties": {
        "enabled": { "type": "boolean" },
        "command": { "type": "array", "items": { "type": "string" } },
        "timeout_ms": { "type": "integer", "minimum": 100, "maximum": 30000 },
        "on_error": { "enum": ["abort", "warn_and_continue", "ignore"] },
        "cwd": { "type": "string" },
        "env": { "type": "object", "additionalProperties": { "type": "string" } }
      }
    },
    "transport": {
      "type": "object",
      "properties": {
        "protocol": { "enum": ["auto", "ndjson", "lsp"] },
        "auto_detect_timeout_ms": { "type": "integer", "minimum": 100, "maximum": 10000 }
      }
    },
    "injectable_tools": {
      "type": "object",
      "properties": {
        "enabled": { "type": "boolean" },
        "tools": { "type": "array", "items": { "type": "string", "pattern": "^hydra_" } },
        "on_collision": { "enum": ["error", "warn", "disable_hydra_tool"] }
      }
    },
    "recorder": {
      "type": "object",
      "properties": {
        "enabled": { "type": "boolean" },
        "buffer_size": { "type": "integer", "minimum": 1, "maximum": 1000 },
        "include_request_bodies": { "type": "boolean" },
        "include_response_bodies": { "type": "boolean" },
        "export_on_crash": { "type": "boolean" },
        "export_path": { "type": "string" }
      }
    },
    "security": {
      "type": "object",
      "properties": {
        "redact_patterns": { "type": "array", "items": { "type": "string" } },
        "redact_replacement": { "type": "string" }
      }
    }
  }
}
```

---

## Configuration Examples

### Minimal Python Server

```json
{
  "version": "1.0",
  "servers": {
    "my-server": {
      "command": "python",
      "args": ["server.py"],
      "cwd": "/Users/dev/projects/my-server"
    }
  }
}
```

### Node Server with Build Hook

```json
{
  "version": "1.0",
  "servers": {
    "my-node-server": {
      "command": "node",
      "args": ["dist/index.js"],
      "cwd": "/Users/dev/projects/my-node-server",
      "watch": {
        "enabled": true,
        "paths": ["./src"],
        "extensions": [".ts", ".js"]
      },
      "pre_restart": {
        "enabled": true,
        "command": ["npm", "run", "build"],
        "timeout_ms": 10000,
        "on_error": "abort"
      }
    }
  }
}
```

### Go Server with Custom Protocol

```json
{
  "version": "1.0",
  "servers": {
    "my-go-server": {
      "command": "/usr/local/bin/my-server",
      "args": ["--port", "8080"],
      "cwd": "/Users/dev/projects/my-go-server",
      "transport": {
        "protocol": "lsp"
      }
    }
  }
}
```

---

**End of Configuration Reference**
