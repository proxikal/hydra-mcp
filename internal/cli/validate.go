package cli

import (
	"fmt"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate Hydra configuration",
		RunE:  validateCommand,
	}

	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func validateCommand(cmd *cobra.Command, args []string) error {
	regPath := viper.GetString("registry")
	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(regPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	for name, srv := range reg.Servers {
		if err := loader.Validate(srv); err != nil {
			return fmt.Errorf("server %s invalid: %w", name, err)
		}
	}

	return nil
}
