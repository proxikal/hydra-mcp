# Hydra: The Self-Healing MCP Supervisor

> **"Cut off one head, two more shall take its place."**

Hydra is a fault-tolerant **Supervisor & Proxy** for Model Context Protocol (MCP) servers. It is built specifically for **AI-Assisted Development**, ensuring that crashes, syntax errors, and "noisy" logs never break the connection between the AI Agent (Claude, Gemini, etc.) and the development server.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Status](https://img.shields.io/badge/status-beta-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.23+-cyan.svg)

---

## 🛑 The Problem

When an AI Agent writes code for an MCP server (e.g., adding a tool to a Python script):
1. The AI saves the file.
2. The server crashes due to a syntax error.
3. **The Connection Dies.** The AI loses its session context, its history, and its ability to fix the error.
4. **The Wallet Bleeds.** You pay tokens to re-explain the context to the AI.

## 🐉 The Hydra Solution

Hydra sits between the AI Client and your MCP Server. It **never dies**, even if the child server does.

```mermaid
graph LR
    AI["AI Agent"] <-->|"Stdio (Safe)"| Hydra["Hydra Proxy"]
    Hydra <-->|"Stdio (Raw)"| Child["MCP Server"]
    
    style Hydra fill:#e6fffa,stroke:#2c7a7b,stroke-width:2px,color:#234e52
    style Child fill:#fff5f5,stroke:#e53e3e,stroke-width:2px,stroke-dasharray: 5 5,color:#742a2a
    style AI fill:#ebf8ff,stroke:#3182ce,stroke-width:2px,color:#2c5282
```

### Key Capabilities

*   **🛡️ Session Persistence:** The AI session survives server crashes. Hydra reports the error as a JSON-RPC message, allowing the AI to read the stack trace and fix the code *without* disconnecting.
*   **🔇 Stdio Sanitization:** Hydra filters `stdout`. If your server prints `console.log("Debug")` or panics with a raw stack trace, Hydra captures it, wraps it in a log message, and prevents it from corrupting the JSON-RPC pipe.
*   **⚡ Optimistic Hot-Reload:** Restarts the server instantly (< 500ms) on file save. No "pre-flight" checks; the crash *is* the feedback.
*   **🧠 Context Resurrection:** When the server restarts, Hydra automatically replays:
    *   `initialize` request
    *   `textDocument/didOpen` & `didChange` (File State)
    *   `resources/subscribe` (Log subscriptions)
    *   `logging/setLevel`
*   **💰 Wallet Guard:** Protects against token bombs.
    *   Truncates massive log messages (max 1KB).
    *   Caps tool outputs at 50KB to prevent accidental dumps.
    *   Rate-limits chatty logs (max 10/sec).

---

## 🚀 Getting Started

### Installation

**From Source:**
```bash
git clone https://github.com/proxikal/hydra.git
cd hydra
go build -o bin/hydra cmd/hydra/main.go
sudo mv bin/hydra /usr/local/bin/
```

**Or using Make:**
```bash
git clone https://github.com/proxikal/hydra.git
cd hydra
make install
```

**Verify Installation:**
```bash
hydra --version
```

> **Note:** Homebrew installation coming in v1.0 release

### Quick Start

**1. Initialize Hydra for Your AI Client**
```bash
hydra init --client claude
# Follow prompts to select which MCP servers to supervise
```

This creates/updates your Claude Desktop config to route servers through Hydra.

**2. Start a Supervised Server**
```bash
hydra run --name my-python-server
```

**3. Check Status**
```bash
hydra status my-python-server
```

**4. View Logs**
```bash
hydra logs my-python-server --follow
```

**5. Manual Restart (if needed)**
```bash
hydra restart my-python-server
```

### Configuration

Hydra uses a two-tier config system:

**Global Registry:** `~/.hydra/config.json`
```json
{
  "servers": {
    "my-python-server": {
      "command": "python",
      "args": ["server.py"],
      "env_file": ".env",
      "watch": {
        "paths": ["./src"],
        "ignore_files": [".gitignore"]
      },
      "max_restarts": 5,
      "restart_window_seconds": 60
    }
  }
}
```

**Local Override (Optional):** `./hydra.json`
```json
{
  "watch": {
    "paths": ["./src", "./lib"]
  }
}
```

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for full schema.

---

## 🛠️ Architecture

Hydra is designed with strict **"AI-Native"** principles:

1.  **Transport Agnostic:** Auto-detects `Content-Length` headers (LSP style) vs NDJSON.
2.  **Cross-Platform:** Uses robust process group management (Tree Kill) to handle zombies on Windows, Linux, and macOS.
3.  **Injectable Tools:** Hydra injects its own tools into the MCP session:
    *   `hydra_restart`: AI can manually trigger a restart.
    *   `hydra_logs`: AI can read the last 50 lines of `stderr`.
    *   `hydra_status`: Check supervisor health.

---

## 🗺️ Project Status

All implementation phases complete:

- ✅ **Phase 1: Foundation** - Transport, Config, Sanitizer
- ✅ **Phase 2: Core Logic** - Supervisor, StateStore, Watcher
- ✅ **Phase 3: Orchestration** - Proxy, Tool Injection, Traffic Recorder
- ✅ **Phase 4: CLI** - Commands, Bootstrap
- ✅ **Phase 5: Hardening** - Chaos Testing, Benchmarks

**Current Version:** Beta (approaching v1.0)

### Roadmap to v1.0
- [ ] Real-world testing with Claude Desktop
- [ ] Performance validation on production workloads
- [ ] Documentation polish
- [ ] Bug fixes from beta feedback

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

**Key Standards:**
- No `fmt.Println` in production code (use `internal/logger`)
- Interface-first pattern (all components)
- Files < 200 lines
- 80%+ test coverage
- TDD mandatory

**Development:**
```bash
git clone https://github.com/proxikal/hydra.git
cd hydra
make test        # Run tests
make lint        # Lint code
make coverage    # Check coverage
```

---

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 📚 Documentation

- [Configuration Reference](docs/CONFIGURATION.md)
- [CLI Reference](docs/CLI_REFERENCE.md)
- [Architecture Details](docs/ARCHITECTURE.md)
- [Security Model](docs/SECURITY.md)
- [Testing Strategy](docs/testing/TESTING_STRATEGY.md)

## 🙏 Acknowledgments

Built with ❤️ for the MCP Community.

Special thanks to all contributors who helped make Hydra production-ready.
