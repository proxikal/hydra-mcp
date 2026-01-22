package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRecoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "recover",
		Short: "Recover a server from FAILED state (use hydra_force_restart tool inside client until IPC channel exists)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("recover requires a running Hydra control channel; use hydra_force_restart tool from your client for now")
		},
	}
}
