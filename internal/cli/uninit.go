package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newUninitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninit",
		Short: "Revert client config changes made by hydra init",
		RunE:  uninitCommand,
	}

	cmd.Flags().String("client", "claude", "Client type (claude, cline, cursor)")
	cmd.Flags().String("config", "", "Client config path (optional, required for cline/cursor)")
	_ = viper.BindPFlag("client", cmd.Flags().Lookup("client"))
	_ = viper.BindPFlag("config", cmd.Flags().Lookup("config"))

	return cmd
}

func uninitCommand(cmd *cobra.Command, args []string) error {
	client := viper.GetString("client")
	configPath, err := resolveClientConfigPath(client, viper.GetString("config"))
	if err != nil {
		return err
	}

	backup := configPath + ".hydra.bak"

	if _, err := os.Stat(backup); os.IsNotExist(err) {
		return fmt.Errorf("no backup found at %s", backup)
	}

	data, err := os.ReadFile(backup)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fmt.Errorf("ensure config dir: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("restore config: %w", err)
	}

	if err := os.Remove(backup); err != nil {
		return fmt.Errorf("remove backup: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Restored %s from backup\n", configPath)
	return nil
}

func resolveClientConfigPath(client, explicit string) (string, error) {
	if explicit != "" {
		return expandHome(explicit), nil
	}

	switch client {
	case "claude":
		return expandHome("~/.config/claude/claude_desktop_config.json"), nil
	case "cline", "cursor":
		return "", fmt.Errorf("--config is required for client %s", client)
	default:
		return "", fmt.Errorf("unsupported client: %s", client)
	}
}
