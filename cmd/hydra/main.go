package main

import (
	"os"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
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
		Use:   "run",
		Short: "Run the supervisor",
		Run: func(cmd *cobra.Command, args []string) {
			// Phase 1 verification: Load config
			loader := config.NewLoader(l)
			cfg, err := loader.Load("hydra.json")
			if err != nil {
				l.Error("Failed to load config", err)
				os.Exit(1)
			}
			l.Info("Loaded configuration", map[string]interface{}{
				"command": cfg.Command,
			})
		},
	}

	rootCmd.AddCommand(runCmd)

	if err := rootCmd.Execute(); err != nil {
		l.Error("Command execution failed", err)
		os.Exit(1)
	}
}