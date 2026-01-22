package cli

import (
	"fmt"
	"path/filepath"

	"github.com/proxikal/hydra/internal/bootstrap"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize client config to route through Hydra",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientType := viper.GetString("client")
			dryRun := viper.GetBool("dry-run")

			log := logger.New("info")
			cfgPath := filepath.Clean(viper.GetString("config"))

			regPath := filepath.Clean(viper.GetString("registry"))
			switch clientType {
			case "claude":
				return bootstrap.ConfigureClaude(cfgPath, regPath, dryRun, log)
			case "cline":
				return bootstrap.ConfigureCline(cfgPath, regPath, dryRun, log)
			case "cursor":
				return bootstrap.ConfigureCursor(cfgPath, regPath, dryRun, log)
			default:
				return fmt.Errorf("unsupported client: %s", clientType)
			}
		},
	}

	cmd.Flags().String("client", "claude", "Client type (claude, cline, cursor)")
	cmd.Flags().Bool("dry-run", false, "Dry run (no file modifications)")
	cmd.Flags().String("config", "", "Client config path (optional)")
	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("client", cmd.Flags().Lookup("client"))
	_ = viper.BindPFlag("dry-run", cmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("config", cmd.Flags().Lookup("config"))
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}
