package cli

import (
	"fmt"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List servers in the Hydra registry",
		RunE:  listCommand,
	}

	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func listCommand(cmd *cobra.Command, args []string) error {
	regPath := viper.GetString("registry")
	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(regPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	for name, srv := range reg.Servers {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s %v\n", name, srv.Command, srv.Args)
	}
	return nil
}
