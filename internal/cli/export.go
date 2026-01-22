package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export recorder crash dump for a server",
		RunE:  exportCommand,
	}

	cmd.Flags().String("name", "", "Server name from registry")
	_ = cmd.MarkFlagRequired("name")
	_ = viper.BindPFlag("name", cmd.Flags().Lookup("name"))

	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	cmd.Flags().String("output", "", "Output path (default: stdout)")
	_ = viper.BindPFlag("output", cmd.Flags().Lookup("output"))

	return cmd
}

func exportCommand(cmd *cobra.Command, args []string) error {
	name := viper.GetString("name")
	regPath := viper.GetString("registry")
	outputPath := viper.GetString("output")

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
	in, err := os.Open(latest)
	if err != nil {
		return fmt.Errorf("open export: %w", err)
	}
	defer func() { _ = in.Close() }()

	if outputPath == "" {
		_, err = io.Copy(cmd.OutOrStdout(), in)
		return err
	}

	outputPath = expandHome(outputPath)
	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Export written to %s (source: %s)\n", outputPath, latest)
	return nil
}
