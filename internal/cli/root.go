package cli

import "github.com/spf13/cobra"

// NewRoot creates the root Cobra command.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "hydra",
		Short: "Hydra MCP supervisor",
	}

	root.AddCommand(
		newRunCommand(),
		newInitCommand(),
		newUninitCommand(),
		newAddCommand(),
		newListCommand(),
		newRemoveCommand(),
		newEditCommand(),
		newValidateCommand(),
		newLogsCommand(),
		newStatusCommand(),
		newRestartCommand(),
		newRecoverCommand(),
		newExportCommand(),
	)

	return root
}

// Execute runs the CLI entrypoint.
func Execute() error {
	return NewRoot().Execute()
}
