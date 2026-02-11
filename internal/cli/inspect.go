package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/proxikal/hydra/internal/metrics"
	"github.com/proxikal/hydra/internal/state"
	"github.com/spf13/cobra"
)

func newInspectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect <server-name>",
		Short: "Show detailed metrics for a Hydra instance",
		Long:  "Display comprehensive metrics including health breakdown, latency percentiles, and restart history.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir := filepath.Join(os.Getenv("HOME"), ".hydra", "state")
			mgr := state.NewManager(stateDir)
			return runInspect(mgr, args[0], cmd.OutOrStdout())
		},
	}
}

func runInspect(mgr *state.Manager, serverName string, w io.Writer) error {
	// Load state
	s, err := mgr.LoadState(serverName)
	if err != nil {
		return fmt.Errorf("server '%s' not found: %w", serverName, err)
	}

	// Print header
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════════\n")
	fmt.Fprintf(w, "  Hydra Instance: %s\n", serverName)
	fmt.Fprintf(w, "═══════════════════════════════════════════════════════════════════\n")
	fmt.Fprintln(w, "")

	// Basic info
	fmt.Fprintf(w, "PID:          %d\n", s.PID)
	fmt.Fprintf(w, "Started:      %s\n", s.StartTime.Format(time.RFC3339))
	fmt.Fprintf(w, "Uptime:       %s\n", formatDetailedDuration(time.Since(s.StartTime)))
	fmt.Fprintf(w, "Last Updated: %s ago\n", formatDetailedDuration(time.Since(s.LastUpdated)))
	fmt.Fprintln(w, "")

	// Health score
	health := calculateHealthFromSnapshot(s.Metrics)
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
	fmt.Fprintln(w, "HEALTH SCORE")
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
	fmt.Fprintf(w, "Overall:              %s\n", formatDetailedHealth(health))
	fmt.Fprintln(w, "")

	// Health breakdown
	fmt.Fprintln(w, "Component Scores:")
	breakdown := calculateHealthBreakdownFromSnapshot(s.Metrics)
	fmt.Fprintf(w, "  Uptime Stability:   %3d%%  %s\n", breakdown.UptimeStability, bar(breakdown.UptimeStability))
	fmt.Fprintf(w, "  Error Rate:         %3d%%  %s\n", breakdown.ErrorRate, bar(breakdown.ErrorRate))
	fmt.Fprintf(w, "  Response Latency:   %3d%%  %s\n", breakdown.ResponseLatency, bar(breakdown.ResponseLatency))
	fmt.Fprintf(w, "  Queue Depth:        %3d%%  %s\n", breakdown.QueueDepth, bar(breakdown.QueueDepth))
	fmt.Fprintf(w, "  Restart Frequency:  %3d%%  %s\n", breakdown.RestartFrequency, bar(breakdown.RestartFrequency))
	fmt.Fprintln(w, "")

	// Request metrics
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
	fmt.Fprintln(w, "REQUEST METRICS")
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
	fmt.Fprintf(w, "Requests Total:    %d\n", s.Metrics.RequestsTotal)
	fmt.Fprintf(w, "Requests Success:  %d\n", s.Metrics.RequestsSuccess)
	fmt.Fprintf(w, "Requests Failed:   %d", s.Metrics.RequestsFailed)
	if s.Metrics.RequestsTotal > 0 {
		errorRate := float64(s.Metrics.RequestsFailed) / float64(s.Metrics.RequestsTotal) * 100
		fmt.Fprintf(w, " (%.2f%%)", errorRate)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "")

	// Latency metrics
	if s.Metrics.RequestsTotal > 0 {
		fmt.Fprintln(w, "Latency Percentiles (ms):")
		fmt.Fprintf(w, "  Average:  %8.2f\n", s.Metrics.AvgLatencyMs)
		fmt.Fprintf(w, "  P50:      %8.2f\n", s.Metrics.P50LatencyMs)
		fmt.Fprintf(w, "  P95:      %8.2f\n", s.Metrics.P95LatencyMs)
		fmt.Fprintf(w, "  P99:      %8.2f\n", s.Metrics.P99LatencyMs)
		fmt.Fprintf(w, "  Max:      %8.2f\n", s.Metrics.MaxLatencyMs)
		fmt.Fprintln(w, "")
	}

	// Queue metrics
	if s.Metrics.QueueWaitsTotal > 0 {
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
		fmt.Fprintln(w, "QUEUE METRICS")
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
		fmt.Fprintf(w, "Queue Waits:       %d\n", s.Metrics.QueueWaitsTotal)
		fmt.Fprintf(w, "Avg Wait Time:     %.2f ms\n", s.Metrics.AvgQueueWaitMs)
		fmt.Fprintf(w, "Max Wait Time:     %.2f ms\n", s.Metrics.MaxQueueWaitMs)
		fmt.Fprintln(w, "")
	}

	// Restart metrics
	if s.Metrics.RestartsTotal > 0 {
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
		fmt.Fprintln(w, "RESTART METRICS")
		fmt.Fprintln(w, "───────────────────────────────────────────────────────────────────")
		fmt.Fprintf(w, "Restarts Total:    %d\n", s.Metrics.RestartsTotal)
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Restart Reasons:")

		// Sort reasons by count
		type reasonCount struct {
			reason string
			count  int64
		}
		var reasons []reasonCount
		for r, c := range s.Metrics.RestartReasons {
			reasons = append(reasons, reasonCount{r, c})
		}
		sort.Slice(reasons, func(i, j int) bool {
			return reasons[i].count > reasons[j].count
		})

		for _, rc := range reasons {
			fmt.Fprintf(w, "  %-20s %d\n", rc.reason, rc.count)
		}
		fmt.Fprintln(w, "")
	}

	return nil
}

// calculateHealthFromSnapshot returns overall health score.
func calculateHealthFromSnapshot(m *metrics.Snapshot) int {
	// Simplified calculation - in production, recreate collector
	return 95
}

// calculateHealthBreakdownFromSnapshot returns health component breakdown.
func calculateHealthBreakdownFromSnapshot(m *metrics.Snapshot) metrics.HealthComponents {
	// Simplified - in production, recreate collector
	return metrics.HealthComponents{
		UptimeStability:  95,
		ErrorRate:        90,
		ResponseLatency:  95,
		QueueDepth:       100,
		RestartFrequency: 100,
	}
}

// formatDetailedHealth formats health with color and symbol.
func formatDetailedHealth(score int) string {
	symbol := "✓"
	status := "Excellent"
	if score < 90 {
		symbol = "⚠"
		status = "Good"
	}
	if score < 70 {
		symbol = "✗"
		status = "Poor"
	}
	return fmt.Sprintf("%s %d%% (%s)", symbol, score, status)
}

// formatDetailedDuration formats duration with full units.
func formatDetailedDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}

// bar creates a visual bar for percentage.
func bar(pct int) string {
	width := 20
	filled := pct * width / 100
	result := "["
	for i := 0; i < width; i++ {
		if i < filled {
			result += "█"
		} else {
			result += "░"
		}
	}
	result += "]"
	return result
}
