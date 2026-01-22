package cli

import (
	"fmt"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a server from the Hydra registry",
		Args:  cobra.ExactArgs(1),
		RunE:  removeCommand,
	}

	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func removeCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	regPath := viper.GetString("registry")
	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(regPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if _, ok := reg.Servers[name]; !ok {
		return fmt.Errorf("server %s not found", name)
	}
	delete(reg.Servers, name)

	if err := config.SaveRegistry(reg, regPath); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}
	return nil
}
