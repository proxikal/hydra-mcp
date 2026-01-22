package main

import (
	"os"
	"time"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/proxy"
	"github.com/proxikal/hydra/internal/recorder"
	"github.com/proxikal/hydra/internal/sanitizer"
	"github.com/proxikal/hydra/internal/statestore"
	"github.com/proxikal/hydra/internal/supervisor"
	"github.com/proxikal/hydra/internal/transport"
	"github.com/spf13/cobra"
)

func main() {
	// Root logger (stderr)
	l := logger.New("info")

	rootCmd := &cobra.Command{
		Use:   "hydra",
		Short: "MCP Hot Reload Supervisor",
		Run: func(cmd *cobra.Command, args []string) {
			l.Info("Hydra is active. Use 'hydra run' to start.")
		},
	}

	runCmd := &cobra.Command{
		Use:   "run <server-name>",
		Short: "Run the supervisor for a named server",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			serverName := args[0]

			// Phase 1 verification: Resolve server config
			loader := config.NewLoader(l)
			cfg, err := loader.ResolveServer(serverName, "hydra.json")
			if err != nil {
				l.Error("Failed to resolve server config", err)
				os.Exit(1)
			}

			if err := loader.Validate(cfg); err != nil {
				l.Error("Invalid configuration", err)
				os.Exit(1)
			}

			l.Info("Loaded configuration", map[string]interface{}{
				"server":  serverName,
				"command": cfg.Command,
			})

			// Build core components (minimal wiring).
			sup := supervisor.NewSupervisor(
				append([]string{cfg.Command}, cfg.Args...),
				cfg.Behavior.MaxRestarts,
				time.Duration(cfg.Behavior.RestartWindowSeconds)*time.Second,
				l,
			)

			store := statestore.New()
			rec := recorder.NewRecorder(recorder.Options{
				Enabled:               cfg.Recorder.Enabled,
				BufferSize:            cfg.Recorder.BufferSize,
				IncludeRequestBodies:  cfg.Recorder.IncludeRequestBodies,
				IncludeResponseBodies: cfg.Recorder.IncludeResponseBodies,
			}, nil, l)
			san := sanitizer.New()

			childTransport := transport.NewStdio(os.Stdin, os.Stdout, l)
			clientTransport := transport.NewStdio(os.Stdin, os.Stdout, l)

			p := proxy.New(
				proxy.Dependencies{
					Logger:              l,
					Sanitizer:           san,
					Supervisor:          sup,
					StateStore:          store,
					Recorder:            rec,
					MaxRestartsInWindow: func() int { return cfg.Behavior.MaxRestarts },
					MaxRestarts:         func() int { return cfg.Behavior.MaxRestarts },
					Child:               childTransport,
					Client:              clientTransport,
				},
				proxy.Options{
					CollisionPolicy:    cfg.InjectableTools.OnCollision,
					CrashExportPath:    cfg.Recorder.ExportPath,
					CrashExportEnabled: cfg.Recorder.ExportOnCrash,
				},
			)

			if err := p.Run(); err != nil {
				l.Error("Proxy terminated", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		l.Error("Command execution failed", err)
		os.Exit(1)
	}
}
