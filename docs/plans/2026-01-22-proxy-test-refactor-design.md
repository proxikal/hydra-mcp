# Proxy Test Refactoring Design

**Date:** 2026-01-22
**Status:** Approved
**Goal:** Split monolithic proxy_test.go (620 lines, 18 tests) into focused test files matching refactored implementation structure

## Problem

The proxy package implementation was refactored into focused files (forwarding.go, message_handler.go, recording.go, state_updates.go, tool_injection.go), but proxy_test.go remains monolithic at 620 lines. Tests are hard to locate and don't match the implementation structure.

## Solution

Split tests across 6 files matching the implementation structure, with shared test utilities extracted.

## File Structure

### test_helpers.go (NEW)
Shared test doubles used across all test files:
- `stubTransport` - Mock transport for child/client
- `stubLogger` - No-op logger for tests
- `mockRecorder` - Mock recorder implementation

### proxy_test.go (TRIMMED - 7 tests)
Core proxy lifecycle & status:
- TestProxyNew
- TestProxyShutdown
- TestProxyShutdownWithoutSupervisor
- TestProxyStatusReflectsSupervisor
- TestProxyStatusCrashLoopCanRecover
- TestProxyRunRecoversFromPanic
- TestLoopErrorTransitionsFailed

### forwarding_test.go (NEW - 2 tests)
- TestProxyForwardToChild
- TestProxyForwardToClient

### message_handler_test.go (NEW - 3 tests)
- TestHandleInitializeResponseCallsReplayBeforeDrain
- TestChildInitializeDrainsQueueAndForwards
- TestSanitizerDropsInvalidChunks

### recording_test.go (NEW - 1 test)
- TestProxyRecord

### state_updates_test.go (NEW - 3 tests)
- TestProxyUpdateStateStore
- TestProxyQueuesDuringStartingAndDrainsAfterRunning
- TestProxyQueueOverflowRespondsWithError

### tool_injection_test.go (NEW - 2 tests)
- TestToolsListMergedAndForwarded
- TestToolsMergeCollisionErrorSendsErrorResponse

## Implementation Strategy

1. **Create test_helpers.go** - Extract shared test doubles
2. **Create new test files incrementally** - Smallest to largest:
   - recording_test.go (1 test)
   - forwarding_test.go (2 tests)
   - tool_injection_test.go (2 tests)
   - message_handler_test.go (3 tests)
   - state_updates_test.go (3 tests)
   - proxy_test.go (trim to 7 tests)
3. **Verify after each file** - Run `go test ./internal/proxy/...`
4. **Final cleanup** - Remove old content from proxy_test.go

## Safety Measures

- Pure code movement, no logic changes
- Incremental verification at each step
- Keep original file until all new files pass
- All files stay in same package (no export issues)

## Benefits

- Tests co-located with implementation concerns
- Easier to find relevant tests
- Reduced cognitive load (smaller files)
- Matches existing package structure
- Maintains all 18 tests with no changes
