package cli

import (
	"fmt"
	"strings"

	"github.com/proxikal/hydra/internal/config"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a server to the Hydra registry",
		Args:  cobra.ExactArgs(1),
		RunE:  addCommand,
	}

	cmd.Flags().String("command", "", "Server command")
	cmd.Flags().String("args", "", "Comma-separated args")
	cmd.Flags().String("cwd", "", "Working directory")
	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")

	_ = viper.BindPFlag("command", cmd.Flags().Lookup("command"))
	_ = viper.BindPFlag("args", cmd.Flags().Lookup("args"))
	_ = viper.BindPFlag("cwd", cmd.Flags().Lookup("cwd"))
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func addCommand(cmd *cobra.Command, args []string) error {
	name := args[0]
	regPath := viper.GetString("registry")
	log := logger.New("info")
	loader := config.NewLoader(log)

	reg, err := loader.LoadRegistry(regPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	server := config.DefaultServerConfig()
	server.Command = viper.GetString("command")
	if server.Command == "" {
		return fmt.Errorf("command is required")
	}
	if argStr := viper.GetString("args"); argStr != "" {
		server.Args = strings.Split(argStr, ",")
	}
	server.CWD = viper.GetString("cwd")

	if err := loader.Validate(server); err != nil {
		return fmt.Errorf("invalid server config: %w", err)
	}

	reg.Servers[name] = server

	if err := config.SaveRegistry(reg, regPath); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	return nil
}
