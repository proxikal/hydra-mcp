package cli

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/proxikal/hydra/internal/discovery"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize client config to route through Hydra",
		Long: `Configures an MCP client to route through Hydra's supervision layer.

If --client is not specified, Hydra will auto-discover installed clients
and prompt you to select one.

Example:
  hydra init                    # Interactive mode
  hydra init --client claude    # Explicit client
  hydra init --dry-run          # Preview changes`,
		RunE: runInit,
	}

	cmd.Flags().String("client", "", "Client type (claude, cline, cursor, etc.)")
	cmd.Flags().Bool("dry-run", false, "Dry run (no file modifications)")
	cmd.Flags().String("config", "", "Client config path (optional)")
	cmd.Flags().String("registry", "~/.hydra/config.json", "Hydra registry path")
	_ = viper.BindPFlag("client", cmd.Flags().Lookup("client"))
	_ = viper.BindPFlag("dry-run", cmd.Flags().Lookup("dry-run"))
	_ = viper.BindPFlag("config", cmd.Flags().Lookup("config"))
	_ = viper.BindPFlag("registry", cmd.Flags().Lookup("registry"))

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	clientType := viper.GetString("client")
	dryRun := viper.GetBool("dry-run")
	cfgPath := viper.GetString("config")
	regPath := viper.GetString("registry")

	log := logger.New("info")

	// If client not specified, run discovery wizard
	if clientType == "" {
		selectedClient, err := runDiscoveryWizard(cmd)
		if err != nil {
			return err
		}
		clientType = selectedClient
	}

	// Clean paths
	if cfgPath != "" {
		cfgPath = filepath.Clean(cfgPath)
	}
	regPath = filepath.Clean(regPath)

	// Bootstrap the client
	if err := bootstrapClient(clientType, cfgPath, regPath, dryRun, log); err != nil {
		return err
	}

	// Success message
	cmd.Printf("\n✓ Successfully configured %s to use Hydra\n", clientType)
	cmd.Println("\nNext steps:")
	cmd.Println("  • Run 'hydra run --name <server>' to start a supervised MCP server")
	cmd.Println("  • Run 'hydra ps' to see running servers")

	return nil
}

func runDiscoveryWizard(cmd *cobra.Command) (string, error) {
	cmd.Println("🔍 Auto-discovering MCP clients...")
	cmd.Println()

	log := logger.New("error") // Quiet for wizard
	scanner := discovery.NewScanner(log)

	discoveries, err := scanner.ScanAll()
	if err != nil {
		return "", fmt.Errorf("discovery failed: %w", err)
	}

	// Resolve conflicts and filter high confidence
	discoveries = discovery.ResolveConflicts(discoveries)
	highConfidence := make([]discovery.Discovery, 0)
	for _, d := range discoveries {
		if d.Confidence >= 0.8 {
			highConfidence = append(highConfidence, d)
		}
	}

	// Handle discovery results
	if len(highConfidence) == 0 {
		cmd.Println("No MCP clients found.")
		cmd.Printf("\nPlease specify a client manually with --client flag.\n")
		cmd.Printf("Supported clients: %s\n", strings.Join(supportedClients(), ", "))
		return "", fmt.Errorf("no clients found")
	}

	if len(highConfidence) == 1 {
		client := highConfidence[0].Client
		cmd.Printf("Found %s with %d server(s)\n", client, highConfidence[0].Servers)
		cmd.Printf("Using %s\n\n", client)
		return client, nil
	}

	// Multiple clients - prompt for selection
	cmd.Println("Multiple clients found:")
	cmd.Println()
	for i, d := range highConfidence {
		cmd.Printf("  %d) %s (%d server(s))\n", i+1, d.Client, d.Servers)
	}
	cmd.Println()

	// Prompt for selection
	choice, err := promptForChoice(cmd, len(highConfidence))
	if err != nil {
		return "", err
	}

	selectedClient := highConfidence[choice-1].Client
	cmd.Printf("\nSelected: %s\n\n", selectedClient)

	return selectedClient, nil
}

func promptForChoice(cmd *cobra.Command, maxChoice int) (int, error) {
	reader := bufio.NewReader(cmd.InOrStdin())

	for {
		cmd.Printf("Select client [1-%d]: ", maxChoice)

		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)

		// Default to 1 if empty
		if input == "" {
			return 1, nil
		}

		choice, err := strconv.Atoi(input)
		if err != nil {
			cmd.Println("Invalid input. Please enter a number.")
			continue
		}

		if choice < 1 || choice > maxChoice {
			cmd.Printf("Invalid choice. Please enter a number between 1 and %d.\n", maxChoice)
			continue
		}

		return choice, nil
	}
}
