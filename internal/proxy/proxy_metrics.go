package proxy

import "time"

// recordRequestComplete records a completed request with its latency and success status.
func (p *proxy) recordRequestComplete(startTime time.Time, success bool) {
	if p.metricsCollector == nil {
		return
	}
	duration := time.Since(startTime)
	p.metricsCollector.RecordRequest(duration, success)
}

// recordQueueWait records the time a request spent waiting in the queue.
func (p *proxy) recordQueueWait(enqueueTime time.Time) {
	if p.metricsCollector == nil {
		return
	}
	duration := time.Since(enqueueTime)
	p.metricsCollector.RecordQueueWait(duration)
}

// recordRestart records a server restart with the given reason.
func (p *proxy) recordRestart(reason string) {
	if p.metricsCollector == nil {
		return
	}
	p.metricsCollector.RecordRestart(reason)
}
