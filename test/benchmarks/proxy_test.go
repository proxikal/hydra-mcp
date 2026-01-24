package benchmarks

import (
	"sort"
	"testing"
	"time"
)

// BenchmarkProxyOverhead measures proxy latency overhead.
// PRD Targets: p50 < 50ms, p99 < 200ms
func BenchmarkProxyOverhead(b *testing.B) {
	bp := newBenchProxy(b, "echo_server.py")
	defer bp.Stop()

	// Send initialize
	msg := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	bp.clientTransport.in <- msg
	<-bp.clientTransport.out
	time.Sleep(100 * time.Millisecond)

	// Benchmark request-response latency
	testMsg := []byte(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	var latencies []time.Duration

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		bp.clientTransport.in <- testMsg

		select {
		case <-bp.clientTransport.out:
			latency := time.Since(start)
			latencies = append(latencies, latency)
		case <-time.After(1 * time.Second):
			b.Fatal("Timeout waiting for response")
		}
	}
	b.StopTimer()

	// Calculate percentiles
	if len(latencies) == 0 {
		b.Fatal("No latency measurements")
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	p50 := latencies[len(latencies)/2]
	p99 := latencies[len(latencies)*99/100]

	b.ReportMetric(float64(p50.Microseconds()), "p50_us")
	b.ReportMetric(float64(p99.Microseconds()), "p99_us")

	// Check against PRD targets
	if p50 > 50*time.Millisecond {
		b.Logf("WARNING: p50 latency %v exceeds 50ms target", p50)
	}
	if p99 > 200*time.Millisecond {
		b.Logf("WARNING: p99 latency %v exceeds 200ms target", p99)
	}
}
