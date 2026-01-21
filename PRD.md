# Product Requirement Document (PRD): Hydra

## 1. Executive Summary
Hydra is a robust, fault-tolerant **Supervisor & Proxy** for Model Context Protocol (MCP) servers. It decouples the AI agent's session from the unstable development lifecycle of the backend server. It ensures that syntax errors, crashes, or panics in the child process never terminate the client connection, allowing for a continuous, "Self-Healing" development workflow.

## 2. Technical Goals & Constraints
- **Zero Friction:** Must work with any standard MCP client (Claude, Gemini, etc.) without client-side plugins.
- **Performance:** < 50ms overhead on requests. < 500ms restart time on file changes.
- **Safety:** Strict sanitization of `stdout` to prevent protocol corruption.
- **Compatibility:** Agnostic to the child server's language (Go, Python, Node, etc.).

## 3. Technology Stack Recommendation

### Core Language
- **Go (Golang) 1.23+**: Chosen for concurrency primitives (channels, goroutines), static typing, and single-binary distribution.

### Critical Libraries (The "No Wheel Re-Creation" List)

#### 1. JSON-RPC 2.0 Handling
* **Candidate:** `github.com/sourcegraph/jsonrpc2`
  * *Why:* Battle-tested in the LSP ecosystem (which MCP mirrors). Handles framing, concurrency, and error codes correctly.
  * *Usage:* Use for parsing traffic to internal state tracking.
* **Alternative (Lightweight):** `github.com/tidwall/gjson` + `github.com/tidwall/sjson`
  * *Why:* Ultra-fast, schema-less parsing. Perfect for a **Proxy** that only needs to peek at `method` and `id` without fully unmarshalling potentially complex/unknown payloads.
  * *Decision:* **`tidwall/gjson`** is preferred for the *Proxy* layer to avoid strict schema coupling and serialization overhead. We only need to *route* messages, not fully understand them (except for specific lifecycle events).

#### 2. File Watching
* **Library:** `github.com/fsnotify/fsnotify`
  * *Why:* The industry standard for cross-platform file system notifications.
  * *Wrapper:* `github.com/radovskyb/watcher` (optional, if recursive polling is needed, but fsnotify is usually sufficient for recursive with a walker).

#### 3. CLI Interface
* **Library:** `github.com/spf13/cobra`
  * *Why:* Standard for Go CLIs. Auto-generates help, flags, and subcommands.

#### 4. Testing & Mocking (Crucial for TDD)
* **Framework:** `github.com/stretchr/testify` (assert, require, suite)
  * *Why:* Replaces Go's verbose error checking with fluent assertions.
* **Mocking:** `github.com/vektra/mockery`
  * *Why:* Auto-generates mocks from interfaces. Essential for testing the `Supervisor` without actually spawning real OS processes every time.

#### 5. Logging (Structured)
* **Library:** `github.com/rs/zerolog`
  * *Why:* Zero-allocation JSON logger. We need high-performance logging to separate "Hydra Logs" from "Traffic Logs".

## 4. Architecture Specifications

### 4.1 Component Diagram
1.  **Transport Interface (`Transport`):** Abstract interface for `Read()` and `Write()`.
    *   *Implementation:* `StdioTransport` (initial), with room for `SSETransport` / `HTTPTransport`.
2.  **Sanitizer:** A robust stream filter that classifies chunks as "Valid JSON-RPC" or "Pollution".
3.  **Supervisor:** Manages the `os.Cmd` process. Handles signals (SIGINT/SIGTERM).
4.  **StateStore:** In-memory store (thread-safe) for `initialize` params and `didChange` history.
5.  **Proxy:** The glue. Routes messages between Transport, StateStore, and Supervisor.

### 4.2 Configuration Schema (`hydra.json`)
To ensure zero-friction setup for AI agents, Hydra will look for a `hydra.json` in the root.

```json
{
  "$schema": "https://hydra.mcp.dev/schema.json",
  "command": "python",
  "args": ["server.py"],
  "environment": {
    "DEBUG": "true",
    "API_KEY": "${ENV:API_KEY}"
  },
  "watch": {
    "paths": ["./src", "./lib"],
    "extensions": [".py", ".json"],
    "ignore": ["**/__pycache__", "**/*.log"]
  },
  "behavior": {
    "debounce_ms": 500,
    "restart_delay_ms": 0,
    "max_restarts": 10
  }
}
```
*Rationale:* A JSON file is easily generated/read by AI. The `$schema` key allows IDEs (and AI) to validate the config structure automatically.

## 5. TDD Strategy (Strict Enforcement)

### Phase 1: The Sanitizer (Pure Function)
*   **Test:** Input raw mixed strings (JSON + logs). Assert only JSON comes out one pipe, logs out the other.
*   **Impl:** Regex/JSON validation.

### Phase 2: The StateStore (In-Memory)
*   **Test:** Send `initialize`, then `didChange`. Assert `GetState()` returns correct replay sequence.
*   **Impl:** Mutex-protected struct.

### Phase 3: The Supervisor (Mocked Process)
*   **Test:** Mock `exec.Command`. Simulate crash (return error). Assert "Restart" event is fired.
*   **Tool:** Use `mockery` to interface the OS layer.

### Phase 4: Integration
*   **Test:** Spawn a real "Echo Server" python script. Send messages. Kill script. Send more messages. Assert recovery.

## 6. Implementation Plan & Milestones

1.  **Project Setup:** Init module, setup `cobra`, setup `golangci-lint`.
2.  **Core Domain:** Define `Message`, `Request`, `Response` structs.
3.  **Transport:** Implement `Reader` and `Writer` with sanitization.
4.  **Supervisor:** Implement process lifecycle (Start, Stop, Restart).
5.  **Proxy:** Wire it all together.

## 7. Open Questions / Risks
- **Transport Framing:** Does the target MCP server use Content-Length headers (LSP style) or plain NDJSON? *Assumption: NDJSON for now, but architecture must support pluggable framers.*
- **Windows Support:** Signal handling (SIGTERM) varies on Windows. Go handles this well, but needs testing.

## 8. Definition of Done
- CI/CD pipeline runs `go test -race ./...`
- 90%+ Code Coverage.
- Passes the "Crash Loop" automated integration test.
