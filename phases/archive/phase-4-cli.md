# Phase 4: CLI (Week 4)

**Goal:** Implement all CLI commands and bootstrap logic.

---

## Tasks

### 1. CLI Package (Part 1: Core Commands)

**Files:**
- `internal/cli/root.go` - Cobra root command
- `internal/cli/run.go` - hydra run
- `internal/cli/init.go` - hydra init
- `internal/cli/uninit.go` - hydra uninit

**Commands:**

#### `hydra run`

**Implementation:**
```go
func runCommand(cmd *cobra.Command, args []string) error {
    serverName := viper.GetString("name")

    // Load config
    cfg, err := config.Load("~/.hydra/config.json")
    if err != nil {
        return err
    }

    // Get server config
    serverCfg, ok := cfg.Servers[serverName]
    if !ok {
        return fmt.Errorf("server %s not found in registry", serverName)
    }

    // Load local override if present
    if localCfg, err := config.Load("./hydra.json"); err == nil {
        serverCfg = config.Merge(serverCfg, localCfg)
    }

    // Create components
    logger := logger.New("info")
    supervisor := supervisor.New(serverCfg, logger)
    stateStore := statestore.New()
    watcher := watcher.New(serverCfg.Watch, logger)
    recorder := recorder.New(serverCfg.Recorder)

    // Create proxy
    proxy := proxy.New(supervisor, stateStore, watcher, recorder, logger)

    // Run
    return proxy.Run()
}
```

#### `hydra init`

**Implementation:**
```go
func initCommand(cmd *cobra.Command, args []string) error {
    clientType := viper.GetString("client")
    dryRun := viper.GetBool("dry-run")

    // Get client config path
    clientConfigPath := getClientConfigPath(clientType)

    // Read client config
    clientConfig, err := readClientConfig(clientConfigPath)
    if err != nil {
        return err
    }

    // Parse MCP server entries
    servers := parseServers(clientConfig)

    // Prompt user to select servers
    selected := promptServerSelection(servers)

    // Create global registry if doesn't exist
    if !fileExists("~/.hydra/config.json") {
        createRegistry()
    }

    // Add servers to registry
    for _, srv := range selected {
        addToRegistry(srv)
    }

    // Backup client config
    if !dryRun {
        backupFile(clientConfigPath)
    }

    // Modify client config (replace command with hydra)
    modifiedConfig := modifyClientConfig(clientConfig, selected)

    // Write modified config
    if !dryRun {
        writeClientConfig(clientConfigPath, modifiedConfig)
    }

    // Display summary
    displaySummary(selected, dryRun)

    return nil
}
```

**Tests:**
- `hydra run` with valid config
- `hydra run` with missing server → error
- `hydra run` merges local override
- `hydra init --client claude` modifies config
- `hydra init --dry-run` doesn't modify files
- `hydra uninit` restores backup

---

### 2. CLI Package (Part 2: Management Commands)

**Files:**
- `internal/cli/add.go` - hydra add
- `internal/cli/list.go` - hydra list
- `internal/cli/remove.go` - hydra remove
- `internal/cli/edit.go` - hydra edit
- `internal/cli/validate.go` - hydra validate

**Commands:**

#### `hydra add`

```go
func addCommand(cmd *cobra.Command, args []string) error {
    serverName := args[0]

    serverCfg := &config.Server{
        Command: viper.GetString("command"),
        Args:    viper.GetStringSlice("args"),
        Cwd:     viper.GetString("cwd"),
        Watch: config.Watch{
            Paths:      viper.GetStringSlice("watch-path"),
            Extensions: viper.GetStringSlice("watch-ext"),
        },
    }

    // Validate
    if err := validateServer(serverCfg); err != nil {
        return err
    }

    // Add to registry
    return registry.Add(serverName, serverCfg)
}
```

#### `hydra list`

```go
func listCommand(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load("~/.hydra/config.json")
    if err != nil {
        return err
    }

    // Get status for each server (if running)
    for name, srv := range cfg.Servers {
        status := getServerStatus(name)  // Query via IPC or pidfile
        displayServerInfo(name, srv, status)
    }

    return nil
}
```

**Tests:**
- `hydra add` creates server entry
- `hydra add` with invalid command → error
- `hydra list` displays all servers
- `hydra remove` deletes server
- `hydra edit` opens in $EDITOR
- `hydra validate` checks config

---

### 3. CLI Package (Part 3: Debugging Commands)

**Files:**
- `internal/cli/logs.go` - hydra logs
- `internal/cli/status.go` - hydra status
- `internal/cli/restart.go` - hydra restart
- `internal/cli/recover.go` - hydra recover
- `internal/cli/export.go` - hydra export-traffic

**Commands:**

#### `hydra logs`

```go
func logsCommand(cmd *cobra.Command, args []string) error {
    serverName := args[0]
    follow := viper.GetBool("follow")
    lines := viper.GetInt("lines")

    // Get log buffer (via IPC or shared memory)
    logs := getServerLogs(serverName, lines)

    // Display logs
    for _, log := range logs {
        fmt.Println(log)
    }

    // Follow if requested
    if follow {
        tailLogs(serverName)
    }

    return nil
}
```

**Tests:**
- `hydra logs` retrieves logs
- `hydra status` returns JSON
- `hydra restart` triggers restart
- `hydra recover` resets FAILED state
- `hydra export-traffic` exports JSON

---

### 4. Bootstrap Logic

**Implementation:**

**Client config parsers:**
- `internal/bootstrap/claude.go` - Parse Claude Desktop config
- `internal/bootstrap/cline.go` - Parse Cline config
- `internal/bootstrap/cursor.go` - Parse Cursor config

**Server selection prompt:**
- Use `github.com/manifoldco/promptui` for interactive prompts

**Config modification:**
- Parse JSON, modify server entries, write back
- Preserve formatting where possible
- Backup before modification

**Tests:**
- Parse Claude config
- Detect server entries
- Modify config correctly
- Backup and restore

---

### 5. Main Entry Point

**File:** `cmd/hydra/main.go`

```go
package main

import (
    "os"
    "github.com/yourorg/hydra/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

---

## Definition of Done (Phase 4)

- [ ] All CLI commands implemented
- [ ] `hydra run` works end-to-end
- [ ] `hydra init` modifies client config correctly
- [ ] `hydra uninit` restores backup
- [ ] All management commands work
- [ ] All debugging commands work
- [ ] CLI tests pass
- [ ] golangci-lint passes
- [ ] Can build binary: `make build`
- [ ] Binary runs on macOS, Linux

---

## Files Created (Phase 4)

```
cmd/hydra/main.go
internal/cli/root.go
internal/cli/run.go
internal/cli/init.go
internal/cli/uninit.go
internal/cli/add.go
internal/cli/list.go
internal/cli/remove.go
internal/cli/edit.go
internal/cli/validate.go
internal/cli/logs.go
internal/cli/status.go
internal/cli/restart.go
internal/cli/recover.go
internal/cli/export.go
internal/bootstrap/claude.go
internal/bootstrap/cline.go
internal/bootstrap/cursor.go
internal/cli/cli_test.go
```

---

## Estimated Time

**5-7 days**
