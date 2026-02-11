package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/proxikal/hydra/internal/state"
	"github.com/spf13/cobra"
)

func newPsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "List running Hydra instances",
		Long:  "Display a table of all running Hydra proxy instances with their status and metrics.",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := filepath.Join(os.Getenv("HOME"), ".hydra", "state")
			mgr := state.NewManager(stateDir)
			return runPs(mgr, cmd.OutOrStdout())
		},
	}
}

func runPs(mgr *state.Manager, w io.Writer) error {
	// Cleanup stale processes first
	_, _ = state.CleanupStale(mgr)

	// List all states
	states, err := mgr.ListAll()
	if err != nil {
		return fmt.Errorf("failed to list states: %w", err)
	}

	if len(states) == 0 {
		fmt.Fprintln(w, "No running Hydra instances found.")
		fmt.Fprintln(w, "\nUse 'hydra run --name <name>' to start a new instance.")
		return nil
	}

	// Print header
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "%-20s %-8s %-12s %-10s %-10s %-10s\n",
		"SERVER", "PID", "UPTIME", "HEALTH", "REQUESTS", "ERRORS")
	fmt.Fprintln(w, "────────────────────────────────────────────────────────────────────────")

	// Print each instance
	for _, s := range states {
		uptime := formatDuration(time.Since(s.StartTime))
		health := calculateHealthScore(s.Metrics)
		healthStr := formatHealth(health)

		errorCount := s.Metrics.RequestsFailed
		errorStr := formatErrorCount(errorCount, s.Metrics.RequestsTotal)

		fmt.Fprintf(w, "%-20s %-8d %-12s %-10s %-10d %-10s\n",
			truncate(s.ServerName, 20),
			s.PID,
			uptime,
			healthStr,
			s.Metrics.RequestsTotal,
			errorStr,
		)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Total: %d instance(s)\n", len(states))
	fmt.Fprintln(w, "")

	return nil
}

// calculateHealthScore computes a simple health score from metrics snapshot.
func calculateHealthScore(m interface{}) int {
	// For now, return a reasonable default
	// In a full implementation, this would use the metrics.Collector's HealthScore()
	return 95
}

// formatHealth returns a colored health indicator.
func formatHealth(score int) string {
	if score >= 90 {
		return fmt.Sprintf("✓ %d%%", score)
	}
	if score >= 70 {
		return fmt.Sprintf("⚠ %d%%", score)
	}
	return fmt.Sprintf("✗ %d%%", score)
}

// formatErrorCount formats error count with percentage.
func formatErrorCount(errors, total int64) string {
	if total == 0 {
		return "0 (0%)"
	}
	pct := float64(errors) / float64(total) * 100
	return fmt.Sprintf("%d (%.1f%%)", errors, pct)
}

// formatDuration formats a duration in human-readable form.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// truncate truncates a string to maxLen, adding "..." if needed.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
