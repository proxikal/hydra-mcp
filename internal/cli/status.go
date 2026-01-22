package cli

import (
	"encoding/json"
	"fmt"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type statusSnapshot struct {
	Name                 string            `json:"name"`
	Command              string            `json:"command"`
	Args                 []string          `json:"args,omitempty"`
	CWD                  string            `json:"cwd,omitempty"`
	RecorderEnabled      bool              `json:"recorder_enabled"`
	RecorderExportPath   string            `json:"recorder_export_path,omitempty"`
	InjectableTools      []string          `json:"injectable_tools,omitempty"`
	OnCollision          string            `json:"on_collision,omitempty"`
	RestartWindowSeconds int               `json:"restart_window_seconds"`
	MaxRestarts          int               `json:"max_restarts"`
	WatchEnabled         bool              `json:"watch_enabled"`
	WatchPaths           []string          `json:"watch_paths,omitempty"`
	SecurityPatterns     int               `json:"security_patterns"`
	Environment          map[string]string `json:"environment,omitempty"`
}

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show configuration status for a server",
		RunE:  statusCommand,
	}

	cmd.Flags().String("name", "", "Server name from registry")
	_ = cmd.MarkFlagRequired("name")
	_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))

	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func statusCommand(cmd *cobra.Command, args []string) error {
	name := viper.GetString("name")
	regPath := viper.GetString("registry")

	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(regPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	srv, ok := reg.Servers[name]
	if !ok {
		return fmt.Errorf("server %s not found", name)
	}

	// Apply defaults to surface merged view.
	merged := config.DefaultServerConfig()
	config.MergeServerConfig(merged, srv)

	out := statusSnapshot{
		Name:                 name,
		Command:              merged.Command,
		Args:                 merged.Args,
		CWD:                  merged.CWD,
		RecorderEnabled:      merged.Recorder.Enabled,
		RecorderExportPath:   merged.Recorder.ExportPath,
		InjectableTools:      merged.InjectableTools.Tools,
		OnCollision:          merged.InjectableTools.OnCollision,
		RestartWindowSeconds: merged.Behavior.RestartWindowSeconds,
		MaxRestarts:          merged.Behavior.MaxRestarts,
		WatchEnabled:         merged.Watch.Enabled,
		WatchPaths:           merged.Watch.Paths,
		SecurityPatterns:     len(merged.Security.RedactPatterns),
		Environment:          merged.Environment,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}

	_, _ = cmd.OutOrStdout().Write(data)
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	return nil
}
