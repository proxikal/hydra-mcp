package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart a server (use hydra_restart tool inside client until IPC channel exists)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("restart command requires a running Hydra control channel; use hydra_restart tool from your client for now")
		},
	}
}
