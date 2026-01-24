package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/proxy"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/security"
	"github.com/proxikal/hydra/internal/statestore"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/proxikal/hydra/internal/watcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run Hydra supervisor for a named server",
		RunE:  runCommand,
	}

	cmd.Flags().String("name", "", "Server name from registry")
	_ = cmd.MarkFlagRequired("name")
	_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))

	cmd.Flags().String("config", "~/.hydra/config.json", "Path to registry config")
	_ = viper.BindPFlag("config", cmd.Flags().Lookup("config"))

	cmd.Flags().String("override", "./hydra.json", "Local override path")
	_ = viper.BindPFlag("override", cmd.Flags().Lookup("override"))

	return cmd
}

func runCommand(cmd *cobra.Command, args []string) error {
	serverName, _ := cmd.Flags().GetString("name")
	registryPath, _ := cmd.Flags().GetString("config")
	overridePath, _ := cmd.Flags().GetString("override")

	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	srvCfg, ok := reg.Servers[serverName]
	if !ok {
		return fmt.Errorf("server %s not found in registry", serverName)
	}

	// Apply defaults
	merged := config.DefaultServerConfig()
	mergeServer := *srvCfg
	config.MergeServerConfig(merged, &mergeServer)

	// Apply local override if present
	if localCfg, err := loader.LoadServerConfig(overridePath); err == nil && localCfg != nil {
		config.MergeServerConfig(merged, localCfg)
	}

	if err := loader.Validate(merged); err != nil {
		return fmt.Errorf("invalid server config: %w", err)
	}

	sup := supervisor.NewSupervisor(
		append([]string{merged.Command}, merged.Args...),
		merged.CWD,
		merged.Behavior.MaxRestarts,
		time.Duration(merged.Behavior.RestartWindowSeconds)*time.Second,
		log,
	)

	// Start supervisor to create stdin/stdout pipes
	// Note: proxy will also call Start(), but it's idempotent
	if err := sup.Start(); err != nil {
		return fmt.Errorf("failed to start supervisor: %w", err)
	}

	store := statestore.New()
	redactor := security.NewRedactorWithReplacement(merged.Security.RedactReplacement)
	rec := recorder.NewRecorder(recorder.Options{
		Enabled:               merged.Recorder.Enabled,
		BufferSize:            merged.Recorder.BufferSize,
		IncludeRequestBodies:  merged.Recorder.IncludeRequestBodies,
		IncludeResponseBodies: merged.Recorder.IncludeResponseBodies,
		RedactPatterns:        merged.Security.RedactPatterns,
	}, redactor, log)

	san := sanitizer.New()

	// Use supervisor's pipes for child transport
	childTransport := transport.NewStdio(sup.Stdout(), sup.Stdin(), log)
	clientTransport := transport.NewStdio(os.Stdin, os.Stdout, log)

	p := proxy.New(
		proxy.Dependencies{
			Logger:              log,
			Sanitizer:           san,
			Supervisor:          sup,
			StateStore:          store,
			Recorder:            rec,
			Redactor:            redactor,
			MaxRestartsInWindow: func() int { return merged.Behavior.MaxRestarts },
			MaxRestarts:         func() int { return merged.Behavior.MaxRestarts },
			Child:               childTransport,
			Client:              clientTransport,
		},
		proxy.Options{
			CollisionPolicy:    merged.InjectableTools.OnCollision,
			CrashExportPath:    merged.Recorder.ExportPath,
			CrashExportEnabled: merged.Recorder.ExportOnCrash,
		},
	)

	// Setup file watcher for hot-reload if enabled
	if merged.Watch.Enabled && len(merged.Watch.Paths) > 0 {
		// Convert relative paths to absolute based on CWD
		watchPaths := make([]string, 0, len(merged.Watch.Paths))
		for _, path := range merged.Watch.Paths {
			absPath := path
			if !filepath.IsAbs(path) {
				absPath = filepath.Join(merged.CWD, path)
			}
			watchPaths = append(watchPaths, absPath)
		}

		// Build ignore patterns
		ignorePatterns := merged.Watch.Ignore

		// Add ignore files patterns (like .gitignore entries)
		// For now, we'll use the ignore list directly
		// TODO: Parse .gitignore files if needed

		// Create watcher with debounce
		debounceMs := merged.Behavior.DebounceMS
		if debounceMs == 0 {
			debounceMs = 300 // Default 300ms
		}
		debounce := time.Duration(debounceMs) * time.Millisecond
		batchWindow := time.Duration(500) * time.Millisecond // 500ms batch window

		w, err := watcher.New(watchPaths, ignorePatterns, debounce, batchWindow, log)
		if err != nil {
			log.Error("failed to create file watcher, hot-reload disabled", err, nil)
		} else {
			if err := w.Start(); err != nil {
				log.Error("failed to start file watcher, hot-reload disabled", err, nil)
			} else {
				log.Info("file watcher started", map[string]interface{}{
					"paths":    watchPaths,
					"debounce": debounce.String(),
				})

				// Start goroutine to handle file change events
				go func() {
					for event := range w.Events() {
						// Check if file matches watched extensions
						if len(merged.Watch.Extensions) > 0 {
							ext := filepath.Ext(event.Path)
							matched := false
							for _, watchExt := range merged.Watch.Extensions {
								if ext == watchExt {
									matched = true
									break
								}
							}
							if !matched {
								continue
							}
						}

						log.Info("file change detected, restarting server", map[string]interface{}{
							"path": event.Path,
						})

						// Trigger supervisor restart
						if err := sup.Restart(); err != nil {
							log.Error("failed to restart server after file change", err, nil)
						}
					}
				}()

				// Cleanup watcher on exit
				defer func() {
					if err := w.Stop(); err != nil {
						log.Error("failed to stop file watcher", err, nil)
					}
				}()
			}
		}
	}

	return p.Run()
}
