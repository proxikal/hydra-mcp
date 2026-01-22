package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type logEntry struct {
	Timestamp string `json:"timestamp"`
	Direction string `json:"direction"`
	Type      string `json:"type"`
}

func newLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show recent recorder crash logs for a server",
		RunE:  logsCommand,
	}

	cmd.Flags().String("name", "", "Server name from registry")
	_ = cmd.MarkFlagRequired("name")
	_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))

	cmd.Flags().Int("lines", 20, "Number of most recent entries to show")
	_ = viper.BindPFlag("lines", cmd.Flags().Lookup("lines"))

	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func logsCommand(cmd *cobra.Command, args []string) error {
	name := viper.GetString("name")
	lines := viper.GetInt("lines")
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

	merged := config.DefaultServerConfig()
	config.MergeServerConfig(merged, srv)

	if !merged.Recorder.ExportOnCrash {
		return fmt.Errorf("recorder crash export is disabled for %s", name)
	}

	globPattern := strings.ReplaceAll(merged.Recorder.ExportPath, "{timestamp}", "*")
	globPattern = expandHome(globPattern)

	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return fmt.Errorf("glob crash exports: %w", err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("no crash exports found at %s", globPattern)
	}

	sort.Slice(matches, func(i, j int) bool {
		infoI, _ := os.Stat(matches[i])
		infoJ, _ := os.Stat(matches[j])
		if infoI == nil || infoJ == nil {
			return matches[i] > matches[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	latest := matches[0]
	data, err := os.ReadFile(latest)
	if err != nil {
		return fmt.Errorf("read export: %w", err)
	}

	var entries []logEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		// If shape changed, just print raw.
		_, _ = cmd.OutOrStdout().Write(data)
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		return nil
	}

	if lines <= 0 || lines > len(entries) {
		lines = len(entries)
	}
	entries = entries[len(entries)-lines:]

	for _, e := range entries {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] %s\n", e.Timestamp, e.Direction, e.Type)
	}
	return nil
}
