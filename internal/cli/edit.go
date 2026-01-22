package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the Hydra registry",
		RunE:  editCommand,
	}

	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func editCommand(cmd *cobra.Command, args []string) error {
	regPath := viper.GetString("registry")
	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(regPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	// Ensure file exists on disk before opening editor.
	if err := config.SaveRegistry(reg, regPath); err != nil {
		return fmt.Errorf("persist registry: %w", err)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	editCmd := exec.Command(editor, regPath)
	editCmd.Stdout = os.Stdout
	editCmd.Stderr = os.Stderr
	editCmd.Stdin = os.Stdin

	if err := editCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Re-load and validate after edit to catch mistakes early.
	updated, err := loader.LoadRegistry(regPath)
	if err != nil {
		return fmt.Errorf("reload registry: %w", err)
	}

	for name, srv := range updated.Servers {
		if err := loader.Validate(srv); err != nil {
			return fmt.Errorf("server %s invalid: %w", name, err)
		}
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Registry updated and validated")
	return nil
}
