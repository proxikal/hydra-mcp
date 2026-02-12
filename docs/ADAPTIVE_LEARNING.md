# Adaptive Learning System

**Hydra's adaptive learning system monitors server behavior and suggests configuration improvements.**

## Overview

Hydra continuously learns from runtime behavior through two components:

1. **Metrics Collector** - Tracks performance metrics and calculates health scores
2. **Adaptive Learner** - Analyzes patterns and suggests configuration optimizations

Both components run automatically and persist data to `~/.hydra/`.

---

## Metrics Collector

### Purpose

The metrics collector tracks real-time performance data and calculates a **0-100 health score** based on five dimensions.

### Tracked Metrics

#### Request Metrics
- **Total requests** - Count of all MCP requests forwarded
- **Success count** - Requests completed successfully
- **Failed count** - Requests that returned errors
- **Success rate** - Percentage of successful requests

#### Latency Metrics (milliseconds)
- **Average latency** - Mean request duration
- **P50 latency** - Median (50th percentile)
- **P95 latency** - 95th percentile
- **P99 latency** - 99th percentile
- **Max latency** - Highest observed latency

#### Queue Metrics
- **Queue wait count** - Number of requests queued during restarts
- **Average queue wait** - Mean time spent in queue
- **Max queue wait** - Longest queue wait time

#### Restart Metrics
- **Total restarts** - Lifetime restart count
- **Restart reasons** - Breakdown by cause (crash, file_change, manual)
- **Restart times** - Timestamps of recent restarts

#### Timing
- **Start time** - When server started
- **Uptime** - Current uptime duration

### Health Score Algorithm

**Overall Score: 0-100** (weighted average of 5 components)

#### 1. Uptime Stability (30% weight)

Measures restart frequency relative to uptime.

```
Score = 100 - (restarts_per_hour * 20)
```

**Targets:**
- 100 points: < 0.1 restarts/hour
- 50 points: 2.5 restarts/hour
- 0 points: ≥ 5 restarts/hour

#### 2. Error Rate (25% weight)

Measures percentage of failed requests.

```
Score = 100 - (error_rate * 200)
```

**Targets:**
- 100 points: < 1% error rate
- 50 points: 25% error rate
- 0 points: ≥ 50% error rate

#### 3. Response Latency (20% weight)

Based on P95 latency (95th percentile).

```
Score = 100 - ((P95 - 100ms) / 49)
```

**Targets:**
- 100 points: P95 < 100ms
- 50 points: P95 = 1000ms
- 0 points: P95 ≥ 5000ms

#### 4. Queue Depth (15% weight)

Measures average queue wait time.

```
Score = 100 - ((avg_wait - 10ms) / 10)
```

**Targets:**
- 100 points: avg wait < 10ms
- 50 points: avg wait = 500ms
- 0 points: avg wait ≥ 1000ms

#### 5. Restart Frequency (10% weight)

Counts recent restarts (last hour or entire uptime if less).

```
Score = 100 - (recent_restarts * 10)
```

**Targets:**
- 100 points: 0 restarts in last hour
- 50 points: 5 restarts in last hour
- 0 points: ≥ 10 restarts in last hour

### Example Health Breakdown

```json
{
  "uptime_stability": 95,    // 30% weight
  "error_rate": 100,          // 25% weight
  "response_latency": 85,     // 20% weight
  "queue_depth": 100,         // 15% weight
  "restart_frequency": 90     // 10% weight
}

// Weighted Score:
// (95 * 0.30) + (100 * 0.25) + (85 * 0.20) + (100 * 0.15) + (90 * 0.10)
// = 28.5 + 25.0 + 17.0 + 15.0 + 9.0
// = 94.5 (rounded to 95)
```

---

## Adaptive Learner

### Purpose

The adaptive learner observes runtime events and suggests configuration improvements based on patterns.

### Observation Types

The learner tracks these events:

1. **restart** - Server restart occurred
2. **queue_full** - Request queue at capacity
3. **high_latency** - Request exceeded latency threshold
4. **crash_loop** - Crash loop detected

### Suggestion Rules

#### Rule 1: Restart Threshold Optimization

**Trigger:** Restart count consistently < 50% of `max_restarts`

**Suggestion:** Lower `max_restarts` to tighter threshold

**Example:**
```
Current: max_restarts = 10
Observed: 2 restarts total after 5 observations
Suggested: max_restarts = 3 (2 * 1.5 buffer)
Confidence: 62% (5 observations / 8)
Reason: "Restart count consistently below threshold"
```

#### Rule 2: Queue Capacity Adjustment

**Trigger:** ≥ 10 `queue_full` observations

**Suggestion:** Increase `queue_size` by 50%

**Example:**
```
Current: queue_size = 100
Observed: 10 queue_full events
Suggested: queue_size = 150
Confidence: 33% (10 / 30)
Reason: "Queue frequently near capacity"
```

#### Rule 3: Latency Spike Mitigation

**Trigger:** > 5 `high_latency` events correlated with restarts

**Suggestion:** Double `debounce_ms` to reduce restart frequency

**Example:**
```
Current: debounce_ms = 500
Observed: 8 high_latency events near restarts
Suggested: debounce_ms = 1000
Confidence: 53% (8 / 15)
Reason: "Latency spikes correlate with restarts"
```

#### Rule 4: Crash Loop Protection

**Trigger:** > 10 `crash_loop` observations

**Suggestion:** Triple `debounce_ms` to slow restart rate

**Example:**
```
Current: debounce_ms = 500
Observed: 15 crash_loop events
Suggested: debounce_ms = 1500
Confidence: 75% (15 / 20)
Reason: "Crash loops detected - increase restart delay"
```

### Confidence Scoring

Suggestions include a **0.0-1.0 confidence score** based on observation count:

- **Low confidence (< 0.5)**: Few observations, suggestion is speculative
- **Medium confidence (0.5-0.8)**: Moderate data, likely beneficial
- **High confidence (> 0.8)**: Strong pattern, highly recommended

**Do NOT auto-apply low-confidence suggestions.**

---

## Integration

### Initialization

```go
// Create metrics collector
collector := metrics.NewCollector()

// Create adaptive learner
learner := adaptive.NewLearner()

// Pass to proxy
proxy.New(proxy.Dependencies{
    MetricsCollector: collector,
    AdaptiveLearner:  learner,
    // ... other deps
})
```

### Recording Metrics

Metrics are recorded automatically by the proxy:

```go
// Request forwarding
start := time.Now()
response, err := forwardRequest(request)
duration := time.Since(start)

collector.RecordRequest(duration, err == nil)

// Queue wait
queueStart := time.Now()
// ... wait for restart to complete
collector.RecordQueueWait(time.Since(queueStart))

// Server restart
collector.RecordRestart("file_change")
```

### Recording Observations

Observations must be recorded manually when events occur:

```go
// Queue at capacity
learner.Observe("queue_full", map[string]interface{}{
    "queue_size": 95,
    "max":        100,
})

// High latency detected
learner.Observe("high_latency", map[string]interface{}{
    "latency_ms":   2500,
    "near_restart": true,
})

// Crash loop triggered
learner.Observe("crash_loop", map[string]interface{}{
    "restarts": 15,
    "window":   60,
})
```

### Analyzing Suggestions

```go
// Get current metrics snapshot
snapshot := collector.Snapshot()

// Get current config values
currentConfig := map[string]int{
    "max_restarts": 10,
    "queue_size":   100,
    "debounce_ms":  500,
}

// Analyze and get suggestions
suggestions := learner.Analyze(snapshot, currentConfig)

// Review suggestions
for _, s := range suggestions {
    fmt.Printf("Parameter: %s\n", s.Parameter)
    fmt.Printf("Current: %d → Suggested: %d\n", s.Current, s.Suggested)
    fmt.Printf("Confidence: %.0f%%\n", s.Confidence * 100)
    fmt.Printf("Reason: %s\n\n", s.Reason)
}
```

---

## Persistence

### Metrics Persistence

Metrics are saved to state files automatically:

```
~/.hydra/state/{server_name}.json
```

State includes metrics snapshot for recovery after restart.

### Learning Data Persistence

Learning observations can be persisted:

```go
data := &adaptive.LearningData{
    ServerName:   "my-server",
    Observations: learner.observations,
    LastAnalyzed: time.Now(),
}

// Save to ~/.hydra/learning/my-server.json
err := adaptive.Save(data, "my-server", "~/.hydra")

// Load on startup
loaded, err := adaptive.Load("my-server", "~/.hydra")
if loaded != nil {
    // Restore observations
}
```

---

## CLI Access

### View Health Score

```bash
$ hydra inspect my-server

Server: my-server
PID: 12345
Uptime: 2h 15m
Health: 95/100 ✓

Health Breakdown:
  Uptime Stability:  95 (30% weight)
  Error Rate:       100 (25% weight)
  Response Latency:  85 (20% weight)
  Queue Depth:      100 (15% weight)
  Restart Frequency: 90 (10% weight)

Metrics:
  Requests: 1523 (1520 success, 3 failed)
  Latency:  P50=45ms, P95=120ms, P99=250ms
  Restarts: 2 (1 file_change, 1 manual)
```

### Injectable Tools

Query metrics from AI client:

```
Use the hydra_status tool
```

**Response:**
```json
{
  "status": "running",
  "pid": 12345,
  "uptime": "2h15m",
  "health": 95,
  "requests_total": 1523,
  "restarts_total": 2
}
```

---

## Prometheus Integration

Metrics can be exposed in Prometheus format:

### Enable Metrics Server

```bash
$ hydra run --name my-server --metrics-port 9090
```

### Scrape Endpoint

```
GET http://localhost:9090/metrics
```

### Example Output

```prometheus
# HELP hydra_requests_total Total number of requests
# TYPE hydra_requests_total counter
hydra_requests_total{server="my-server"} 1523

# HELP hydra_requests_success Successful requests
# TYPE hydra_requests_success counter
hydra_requests_success{server="my-server"} 1520

# HELP hydra_requests_failed Failed requests
# TYPE hydra_requests_failed counter
hydra_requests_failed{server="my-server"} 3

# HELP hydra_latency_ms Request latency in milliseconds
# TYPE hydra_latency_ms summary
hydra_latency_ms{server="my-server",quantile="0.5"} 45
hydra_latency_ms{server="my-server",quantile="0.95"} 120
hydra_latency_ms{server="my-server",quantile="0.99"} 250

# HELP hydra_health_score Overall health score (0-100)
# TYPE hydra_health_score gauge
hydra_health_score{server="my-server"} 95

# HELP hydra_health_component Health component scores
# TYPE hydra_health_component gauge
hydra_health_component{server="my-server",component="uptime_stability"} 95
hydra_health_component{server="my-server",component="error_rate"} 100
hydra_health_component{server="my-server",component="response_latency"} 85
hydra_health_component{server="my-server",component="queue_depth"} 100
hydra_health_component{server="my-server",component="restart_frequency"} 90

# HELP hydra_restarts_total Total number of restarts
# TYPE hydra_restarts_total counter
hydra_restarts_total{server="my-server"} 2

# HELP hydra_restarts_by_reason Restart count by reason
# TYPE hydra_restarts_by_reason counter
hydra_restarts_by_reason{server="my-server",reason="file_change"} 1
hydra_restarts_by_reason{server="my-server",reason="manual"} 1
```

---

## Best Practices

### 1. Monitor Health Trends

Don't react to single health dips. Track trends over time:

```bash
# Check health hourly
*/60 * * * * hydra inspect my-server | grep "Health:"
```

### 2. Review Suggestions Weekly

Analyze suggestions periodically:

```go
// In your monitoring system
suggestions := learner.Analyze(snapshot, currentConfig)

for _, s := range suggestions {
    if s.Confidence > 0.7 {
        // Log high-confidence suggestions for review
        log.Printf("HIGH CONFIDENCE: %s", s.Reason)
    }
}
```

### 3. Apply Changes Gradually

When applying suggestions:

1. Apply ONE parameter at a time
2. Monitor health for 24-48 hours
3. Revert if health decreases
4. Apply next suggestion

### 4. Preserve Learning Data

Backup learning data before major changes:

```bash
cp -r ~/.hydra/learning ~/.hydra/learning.backup
```

### 5. Reset After Major Changes

Reset observations after infrastructure changes:

```bash
rm ~/.hydra/learning/my-server.json
hydra restart my-server
```

---

## Troubleshooting

### Health Score is Low

**Check health breakdown:**
```bash
hydra inspect my-server
```

**Focus on lowest component:**
- **Uptime Stability < 50**: Reduce restart frequency (check logs for crash causes)
- **Error Rate < 50**: Investigate why requests are failing (check child server logs)
- **Response Latency < 50**: Optimize child server or increase resources
- **Queue Depth < 50**: Increase queue_size or reduce restart frequency
- **Restart Frequency < 50**: Address root cause of frequent restarts

### No Suggestions Generated

Learner needs minimum observations:

- **Restart threshold**: ≥ 5 restart observations
- **Queue capacity**: ≥ 10 queue_full observations
- **Latency spikes**: ≥ 6 high_latency observations
- **Crash loops**: ≥ 11 crash_loop observations

**Record observations explicitly:**
```go
learner.Observe("restart", map[string]interface{}{})
```

### Suggestions Contradict Each Other

This can happen when multiple patterns overlap:

**Resolution:**
1. Sort by confidence (highest first)
2. Apply highest-confidence suggestion
3. Wait for new data
4. Re-analyze

---

## Performance Impact

### Metrics Collector

- **Memory**: ~1MB per 1000 requests (latency/queue arrays)
- **CPU**: < 0.1% (mutex-protected updates)
- **Latency overhead**: < 10μs per request (time.Now() calls)

### Adaptive Learner

- **Memory**: ~100 bytes per observation (typical: < 1MB total)
- **CPU**: Negligible (analysis runs on-demand, not per-request)

**Both components are optimized for low overhead.**

---

## Future Enhancements

Potential improvements (not yet implemented):

1. **Auto-apply high-confidence suggestions** (with rollback)
2. **Time-series anomaly detection** (detect unusual patterns)
3. **Multi-server correlation** (learn from fleet behavior)
4. **ML-based predictions** (predict crashes before they occur)

---

## Reference

### Metrics Collector Interface

```go
type MetricsCollector interface {
    RecordRequest(duration time.Duration, success bool)
    RecordQueueWait(duration time.Duration)
    RecordRestart(reason string)
    Snapshot() *Snapshot
    HealthScore() int
    HealthBreakdown() HealthComponents
}
```

### Adaptive Learner Interface

```go
type Learner struct {
    observations []Observation
}

func (l *Learner) Observe(event string, ctx map[string]interface{})
func (l *Learner) Analyze(snapshot *Snapshot, config map[string]int) []Suggestion
```

### Suggestion Structure

```go
type Suggestion struct {
    Parameter  string  // "max_restarts", "queue_size", "debounce_ms"
    Current    int     // Current value
    Suggested  int     // Suggested value
    Confidence float64 // 0.0-1.0 confidence score
    Reason     string  // Human-readable explanation
}
```

---

**For implementation details, see:**
- `internal/metrics/collector.go` - Metrics implementation
- `internal/adaptive/learner.go` - Learning algorithm
- `internal/proxy/proxy.go` - Integration point
