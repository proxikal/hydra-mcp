package benchmarks

import (
	"testing"
	"time"
)

// TestMemoryLeakRequests verifies no memory leak during request/response cycles.
// PRD Target: Memory increase < 100MB RSS after sustained load
func TestMemoryLeakRequests(t *testing.T) {
	b := &testing.B{}
	bp := newBenchProxy(b, "echo_server.py")
	defer bp.Stop()

	// Initialize
	initMsg := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	bp.clientTransport.in <- initMsg
	<-bp.clientTransport.out
	time.Sleep(100 * time.Millisecond)

	// Capture baseline
	before := captureMemory()

	// Simulate sustained load (1000 request/response cycles)
	iterations := 1000
	for i := 0; i < iterations; i++ {
		testMsg := []byte(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
		bp.clientTransport.in <- testMsg
		select {
		case <-bp.clientTransport.out:
		case <-time.After(100 * time.Millisecond):
		}

		// Periodic GC to clear truly dead objects
		if i%100 == 0 {
			captureMemory()
		}
	}

	// Final measurement
	after := captureMemory()
	increaseMB := memoryIncrease(before, after)
	goroutines := goroutineIncrease(before, after)

	t.Logf("Memory: %.2fMB increase, %d goroutines leaked (after %d iterations)",
		increaseMB, goroutines, iterations)

	// Hard failure if targets exceeded
	maxMemoryMB := 100.0
	if increaseMB > maxMemoryMB {
		t.Fatalf("FAIL: Memory increase %.2fMB exceeds %dMB target", increaseMB, int(maxMemoryMB))
	}

	maxGoroutines := 10
	if goroutines > maxGoroutines {
		t.Fatalf("FAIL: Goroutine leak detected: %d new goroutines (max %d)", goroutines, maxGoroutines)
	}

	t.Logf("PASS: Memory %.2fMB < %dMB, Goroutines %d < %d",
		increaseMB, int(maxMemoryMB), goroutines, maxGoroutines)
}

// TestMemoryLeakConcurrent verifies no memory leak under concurrent load.
func TestMemoryLeakConcurrent(t *testing.T) {
	b := &testing.B{}
	bp := newBenchProxy(b, "echo_server.py")
	defer bp.Stop()

	// Initialize
	initMsg := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	bp.clientTransport.in <- initMsg
	<-bp.clientTransport.out
	time.Sleep(100 * time.Millisecond)

	// Capture baseline
	before := captureMemory()

	// Concurrent requests (500 iterations to keep test fast)
	iterations := 500
	concurrency := 5

	done := make(chan bool, concurrency)
	for w := 0; w < concurrency; w++ {
		go func(workerID int) {
			for i := 0; i < iterations/concurrency; i++ {
				testMsg := []byte(`{"jsonrpc":"2.0","id":3,"method":"ping"}`)
				bp.clientTransport.in <- testMsg
				select {
				case <-bp.clientTransport.out:
				case <-time.After(200 * time.Millisecond):
				}
			}
			done <- true
		}(w)
	}

	// Wait for all workers
	for w := 0; w < concurrency; w++ {
		<-done
	}

	// Final measurement
	after := captureMemory()
	increaseMB := memoryIncrease(before, after)
	goroutines := goroutineIncrease(before, after)

	t.Logf("Concurrent: %.2fMB increase, %d goroutines leaked", increaseMB, goroutines)

	// Hard failure
	maxMemoryMB := 100.0
	if increaseMB > maxMemoryMB {
		t.Fatalf("FAIL: Memory increase %.2fMB exceeds %dMB target", increaseMB, int(maxMemoryMB))
	}

	maxGoroutines := 15
	if goroutines > maxGoroutines {
		t.Fatalf("FAIL: Goroutine leak: %d new goroutines (max %d)", goroutines, maxGoroutines)
	}

	t.Logf("PASS: Concurrent memory %.2fMB < %dMB, Goroutines %d < %d",
		increaseMB, int(maxMemoryMB), goroutines, maxGoroutines)
}
