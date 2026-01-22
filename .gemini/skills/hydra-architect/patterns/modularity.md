# Hydra Pattern: Modularity & Structure

**Rule:** Small Packages, Small Files.
**Reason:** Context efficiency. Agents shouldn't read 500 lines to change 1 function.

## Package Rules

1.  **Directory = Package**
    *   `internal/proxy` is package `proxy`.
    *   `internal/config` is package `config`.
    *   NO nested packages inside `internal` unless justified.

2.  **File Limits**
    *   **Max 200 lines** per file.
    *   If a file grows > 200 lines, split it by responsibility:
        *   `manager.go` (Main logic)
        *   `manager_lifecycle.go` (Start/Stop)
        *   `manager_test.go` (Tests)

3.  **Dependency Injection**
    *   Pass dependencies in the Constructor.
    *   NEVER use `config.GlobalConfig` or singletons.

## Directory Map
```
cmd/hydra/       # Entry point only
internal/
  transport/     # IO logic
  protocol/      # Parsing logic
  supervisor/    # Process logic
  proxy/         # Routing logic
```
