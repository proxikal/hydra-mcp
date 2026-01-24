# Changelog

All notable changes to Hydra will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - TBD

### Added
- **Core Supervision**: Fault-tolerant MCP server supervision with crash recovery
- **Hot-Reload**: Automatic server restart on file changes with configurable watch paths
- **Session Continuity**: AI client sessions survive server crashes via state replay
- **STDIO Sanitization**: Filters stdout pollution to prevent JSON-RPC corruption
- **State Resurrection**: Automatic replay of initialize, didOpen, subscribe requests
- **Request Queueing**: Queue up to 100 requests during restart (30s TTL)
- **Crash Loop Protection**: Detect and halt after 5 restarts in 60 seconds
- **Injectable Tools**:
  - `hydra_restart` - Manual server restart
  - `hydra_status` - Supervisor status
  - `hydra_logs` - View child stderr logs (last 50 lines)
  - `hydra_force_restart` - Override crash loop protection
- **Wallet Guard**:
  - Log truncation (1000 chars max)
  - Payload limiting (50KB max)
  - Rate limiting (10 logs/sec)
- **Traffic Recorder**: Debug mode for JSON-RPC traffic inspection
- **CLI Commands**:
  - `hydra run` - Start supervised server
  - `hydra init` - Bootstrap client configuration
  - `hydra add` - Register new server
  - `hydra list` - List registered servers
  - `hydra remove` - Unregister server
  - `hydra logs` - View server logs
  - `hydra status` - Check server status
  - `hydra restart` - Manual restart
  - `hydra recover` - Reset crash counter
- **Configuration System**:
  - Global registry (`~/.hydra/config.json`)
  - Local overrides (`./hydra.json`)
  - Pre-restart hooks
  - Custom environment variables
  - File watch patterns with gitignore support
- **Security Features**:
  - Secret redaction patterns
  - Process group tree kill (no zombies)
  - Hook timeout protection
  - Config validation

### Performance
- Proxy latency: p50 < 50ms, p99 < 200ms
- Restart time: p50 < 500ms, p99 < 2s
- Memory: < 100MB after 1000 restarts
- CPU idle: < 1%

### Testing
- 80%+ overall test coverage
- 90%+ coverage on critical paths (proxy, supervisor)
- Race detector clean
- Chaos tests (slow startup, chatty servers, mass file changes)
- Benchmark tests (latency, restart speed, memory leaks)
- Integration tests (crash loops, queueing, state resurrection)

### Documentation
- Complete PRD with architecture details
- Configuration reference
- CLI command reference
- Security model documentation
- Testing strategy guides
- Contributing guidelines

---

**Full Changelog**: https://github.com/proxikal/hydra/commits/v1.0.0
