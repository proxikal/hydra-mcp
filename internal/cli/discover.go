package cli

import (
	"fmt"
	"strings"

	"github.com/proxikal/hydra/internal/discovery"
	"github.com/proxikal/hydra/internal/logger"
	"github.com/spf13/cobra"
)

func newDiscoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Auto-detect installed MCP clients",
		Long: `Scans for installed MCP clients (Claude, Cline, Cursor, etc.) and displays
their configurations with confidence scores.

Example:
  hydra discover`,
		RunE: runDiscover,
	}
}

func runDiscover(cmd *cobra.Command, args []string) error {
	log := logger.New("info")
	scanner := discovery.NewScanner(log)

	// Scan all clients
	discoveries, err := scanner.ScanAll()
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Resolve conflicts (keep highest confidence per client)
	discoveries = discovery.ResolveConflicts(discoveries)

	// Filter to high confidence discoveries
	highConfidence := make([]discovery.Discovery, 0)
	lowConfidence := make([]discovery.Discovery, 0)

	for _, d := range discoveries {
		if d.Confidence >= 0.8 {
			highConfidence = append(highConfidence, d)
		} else if d.Confidence >= 0.5 {
			lowConfidence = append(lowConfidence, d)
		}
	}

	// Display results
	if len(highConfidence) == 0 && len(lowConfidence) == 0 {
		cmd.Println("No MCP clients found.")
		cmd.Println("\nSupported clients: claude, cline, cursor, claude-cli, continue, cursor-composer")
		return nil
	}

	cmd.Println("\n🔍 Discovered MCP Clients:")

	// Print table header
	printTableHeader(cmd)

	// Print high confidence discoveries
	for _, d := range highConfidence {
		printDiscoveryRow(cmd, d, false)
	}

	// Print low confidence discoveries with warning
	if len(lowConfidence) > 0 {
		cmd.Println()
		for _, d := range lowConfidence {
			printDiscoveryRow(cmd, d, true)
		}
	}

	// Print summary
	cmd.Println()
	if len(highConfidence) > 0 {
		cmd.Printf("✓ Found %d client(s) with valid configurations\n", len(highConfidence))
	}
	if len(lowConfidence) > 0 {
		cmd.Printf("⚠ Found %d client(s) with incomplete configurations\n", len(lowConfidence))
	}

	cmd.Println("\nNext steps:")
	cmd.Println("  • Run 'hydra init' to configure Hydra")
	cmd.Println("  • Run 'hydra add <name>' to add a new MCP server")

	return nil
}

func printTableHeader(cmd *cobra.Command) {
	// Calculate column widths
	clientWidth := 20
	pathWidth := 50
	serversWidth := 10
	confidenceWidth := 12

	// Print header
	cmd.Printf("%-*s %-*s %-*s %-*s\n",
		clientWidth, "CLIENT",
		pathWidth, "PATH",
		serversWidth, "SERVERS",
		confidenceWidth, "CONFIDENCE")

	// Print separator
	cmd.Println(strings.Repeat("-", clientWidth+pathWidth+serversWidth+confidenceWidth+3))
}

func printDiscoveryRow(cmd *cobra.Command, d discovery.Discovery, warning bool) {
	clientWidth := 20
	pathWidth := 50
	serversWidth := 10
	confidenceWidth := 12

	// Truncate path if too long
	displayPath := d.ConfigPath
	if len(displayPath) > pathWidth {
		displayPath = "..." + displayPath[len(displayPath)-pathWidth+3:]
	}

	// Format confidence as percentage
	confidenceStr := fmt.Sprintf("%.0f%%", d.Confidence*100)
	if warning {
		confidenceStr = "⚠ " + confidenceStr
	}

	cmd.Printf("%-*s %-*s %-*d %-*s\n",
		clientWidth, d.Client,
		pathWidth, displayPath,
		serversWidth, d.Servers,
		confidenceWidth, confidenceStr)
}
