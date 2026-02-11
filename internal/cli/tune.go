package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/proxikal/hydra/internal/adaptive"
	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/proxikal/hydra/internal/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newTuneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tune <server-name>",
		Short: "Show adaptive learning suggestions",
		Long: `Analyzes server behavior and suggests configuration improvements
based on observed patterns.

Example:
  hydra tune weather              # Show suggestions
  hydra tune weather --apply      # Apply suggestions`,
		Args: cobra.ExactArgs(1),
		RunE: runTune,
	}

	cmd.Flags().Bool("apply", false, "Apply suggestions to config")
	_ = viper.BindPFlag("apply", cmd.Flags().Lookup("apply"))

	return cmd
}

func runTune(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	apply := viper.GetBool("apply")
	log := logger.New("info")

	// Determine state directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	stateDir := filepath.Join(homeDir, ".hydra/state")

	// Load learning data
	learningDir := filepath.Join(homeDir, ".hydra")
	data, err := adaptive.Load(serverName, learningDir)
	if err != nil {
		return fmt.Errorf("failed to load learning data: %w", err)
	}

	if data == nil {
		cmd.Println("\nNo learning data available for this server yet.")
		cmd.Println("Run the server for a while to collect observations.")
		return nil
	}

	// Load current metrics from state
	stateMgr := state.NewManager(stateDir)
	serverState, err := stateMgr.LoadState(serverName)
	if err != nil || serverState == nil {
		cmd.Println("\nServer state not found. Is the server running?")
		return nil
	}

	// Load current config
	registryPath := filepath.Join(homeDir, ".hydra/config.json")
	loader := config.NewLoader(log)
	registry, err := loader.LoadRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	serverConfig, exists := registry.Servers[serverName]
	if !exists {
		return fmt.Errorf("server %s not found in registry", serverName)
	}

	// Reconstruct learner from saved observations
	learner := adaptive.NewLearner()
	for _, obs := range data.Observations {
		learner.Observe(obs.Event, obs.Context)
	}

	// Get current config values
	currentConfig := map[string]int{
		"max_restarts": serverConfig.Behavior.MaxRestarts,
		"queue_size":   100, // TODO: Get from options if stored
		"debounce_ms":  serverConfig.Behavior.DebounceMS,
	}

	// Analyze and get suggestions
	suggestions := learner.Analyze(serverState.Metrics, currentConfig)

	// Filter to high confidence suggestions (> 60%)
	highConfidence := make([]adaptive.Suggestion, 0)
	for _, s := range suggestions {
		if s.Confidence > 0.6 {
			highConfidence = append(highConfidence, s)
		}
	}

	// Display results
	if len(highConfidence) == 0 {
		cmd.Println("\n✓ No configuration adjustments recommended")
		cmd.Println("  Current settings appear optimal for observed behavior.")
		return nil
	}

	cmd.Println("\n🔧 Adaptive Learning Suggestions:")

	for i, s := range highConfidence {
		cmd.Printf("%d. %s\n", i+1, s.Parameter)
		cmd.Printf("   Current:    %d\n", s.Current)
		cmd.Printf("   Suggested:  %d\n", s.Suggested)
		cmd.Printf("   Confidence: %.0f%%\n", s.Confidence*100)
		cmd.Printf("   Reason:     %s\n\n", s.Reason)
	}

	// Apply if requested
	if apply {
		cmd.Println("Applying suggestions...")

		for _, s := range highConfidence {
			switch s.Parameter {
			case "max_restarts":
				serverConfig.Behavior.MaxRestarts = s.Suggested
			case "debounce_ms":
				serverConfig.Behavior.DebounceMS = s.Suggested
			case "queue_size":
				// TODO: Apply queue_size when Options are stored in config
				cmd.Printf("⚠ Cannot apply queue_size change (requires server restart)\n")
			}
		}

		// Save updated config
		if err := config.SaveRegistry(registry, registryPath); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		cmd.Println("\n✓ Configuration updated")
		cmd.Println("  Restart the server for changes to take effect:")
		cmd.Printf("    hydra restart %s\n", serverName)
	} else {
		cmd.Println("To apply these suggestions, run:")
		cmd.Printf("  hydra tune %s --apply\n", serverName)
	}

	return nil
}
