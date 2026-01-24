package benchmarks

import (
	"runtime"
	"testing"
	"time"
)

// TestGoroutineLeakBasic verifies no goroutine leaks during normal operation.
func TestGoroutineLeakBasic(t *testing.T) {
	b := &testing.B{}
	bp := newBenchProxy(b, "echo_server.py")
	defer bp.Stop()

	// Initialize
	initMsg := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	bp.clientTransport.in <- initMsg
	<-bp.clientTransport.out
	time.Sleep(100 * time.Millisecond)

	// Capture baseline goroutine count
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	baselineGoroutines := runtime.NumGoroutine()

	// Perform operations
	for i := 0; i < 100; i++ {
		testMsg := []byte(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
		bp.clientTransport.in <- testMsg
		select {
		case <-bp.clientTransport.out:
		case <-time.After(100 * time.Millisecond):
		}
	}

	// Allow time for goroutines to clean up
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Measure final goroutine count
	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - baselineGoroutines

	t.Logf("Goroutines: baseline=%d final=%d leaked=%d",
		baselineGoroutines, finalGoroutines, leaked)

	// Allow small variance for runtime goroutines
	maxLeaked := 5
	if leaked > maxLeaked {
		t.Fatalf("FAIL: Goroutine leak detected: %d goroutines leaked (max %d)",
			leaked, maxLeaked)
	}

	t.Logf("PASS: Goroutine leak check passed (%d <= %d)", leaked, maxLeaked)
}

// TestGoroutineLeakStress verifies no goroutine leaks under stress.
func TestGoroutineLeakStress(t *testing.T) {
	b := &testing.B{}
	bp := newBenchProxy(b, "echo_server.py")
	defer bp.Stop()

	// Initialize
	initMsg := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	bp.clientTransport.in <- initMsg
	<-bp.clientTransport.out
	time.Sleep(100 * time.Millisecond)

	// Capture baseline
	runtime.GC()
	time.Sleep(50 * time.Millisecond)
	before := captureMemory()

	// Stress test with concurrent workers
	iterations := 200
	workers := 10
	done := make(chan bool, workers)

	for w := 0; w < workers; w++ {
		go func() {
			for i := 0; i < iterations/workers; i++ {
				testMsg := []byte(`{"jsonrpc":"2.0","id":3,"method":"ping"}`)
				bp.clientTransport.in <- testMsg
				select {
				case <-bp.clientTransport.out:
				case <-time.After(200 * time.Millisecond):
				}
			}
			done <- true
		}()
	}

	// Wait for all workers
	for w := 0; w < workers; w++ {
		<-done
	}

	// Allow cleanup
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	// Measure
	after := captureMemory()
	leaked := goroutineIncrease(before, after)

	t.Logf("Stress test: %d goroutines leaked after %d concurrent operations",
		leaked, iterations)

	// Allow more variance under stress
	maxLeaked := 20
	if leaked > maxLeaked {
		t.Fatalf("FAIL: Goroutine leak under stress: %d leaked (max %d)",
			leaked, maxLeaked)
	}

	t.Logf("PASS: Stress test goroutine check passed (%d <= %d)", leaked, maxLeaked)
}
