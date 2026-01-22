# Config Test Refactoring Design

**Date:** 2026-01-22
**Status:** Approved
**Goal:** Split monolithic config_test.go (556 lines, 14 tests) into focused test files matching implementation structure

## Problem

The config package has well-structured implementation files (loader.go, registry.go, envsubst.go, config.go) but config_test.go remains monolithic at 556 lines with 14 test functions. Tests are hard to locate and don't match the implementation structure.

## Solution

Split tests across 4 files matching the implementation structure, following Go conventions by keeping tests next to implementation files in the same directory.

## File Structure

### envsubst_test.go (NEW - 2 tests)
Tests for envsubst.go:
- TestSubstituteEnvVars
- TestSubstituteEnvVarsInConfig

### config_test.go (TRIMMED - 3 tests)
Tests for config.go:
- TestMergeServerConfig
- TestDefaultServerConfig
- TestMerge_EdgeCases

### registry_test.go (NEW - 5 tests)
Tests for registry.go:
- TestSaveRegistry
- TestRegistryOperations
- TestDefaultRegistry
- TestSaveRegistry_EdgeCases
- TestAddServer_NilMap

### loader_test.go (NEW - 4 tests)
Tests for loader.go:
- TestLoader_LoadRegistry
- TestLoader_LoadServerConfig
- TestLoader_Validate
- TestLoader_ResolveServer_Integration

## Shared Test Utilities

**Strategy:** Only create test_helpers.go if genuinely shared utilities emerge during the split. Config tests appear more self-contained than proxy tests.

Most tests use:
- Standard `mocks.NewLogger(t)` (already centralized)
- Test-specific `t.TempDir()` and file setup
- Standard assertions

**Decision:** Don't create test_helpers.go preemptively. Add only if 3+ test files need the same utility.

## Implementation Strategy

1. **Start with smallest files first:**
   - envsubst_test.go (2 tests)
   - config_test.go (3 tests, trim original)
   - registry_test.go (5 tests)
   - loader_test.go (4 tests)

2. **For each file:**
   - Copy tests from config_test.go
   - Verify imports
   - Run `go test ./internal/config/...`
   - Remove from original config_test.go
   - Verify again

3. **Final verification:**
   - Run full test suite
   - Confirm all 14 tests pass
   - Check line count reduction

## Safety Measures

- Pure code movement, no logic changes
- Incremental verification at each step
- Keep original until all new files pass
- All files stay in `internal/config` package (Go convention)

## Benefits

- Tests co-located with implementation
- Follows Go conventions (test files next to implementation)
- Easier to find relevant tests
- Reduced cognitive load (smaller files)
- Expected 73% reduction in main test file (556 → ~120-150 lines)

## Why Not a Tests Subfolder?

Considered `/internal/config/tests/` but rejected because:
- Breaks Go conventions (99% of Go projects use same-directory tests)
- IDEs won't auto-pair test/implementation files
- Makes `go test` commands less intuitive
- Harder for developers to find which test covers which code

Professional Go code keeps tests next to implementation.
