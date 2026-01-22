# Hydra Pattern: TDD & Mocking

**Rule:** No implementation without a failing test.
**Reason:** Safety. We are building a Supervisor; reliability is paramount.

## Testing Strategy

1.  **Mockery First**
    *   Run `make mocks` to generate mocks for your new Interface.
    *   Use `internal/mocks` package.

2.  **Testify Suite**
    *   Use `suite.Suite` for complex tests.
    *   Use `assert` for simple checks.

```go
func TestManager(t *testing.T) {
    mockLogger := new(mocks.Logger)
    mgr := NewManager(mockLogger)
    
    // Expectation
    mockLogger.On("Info", "started").Return()
    
    // Action
    err := mgr.Start()
    
    // Assertion
    assert.NoError(t, err)
    mockLogger.AssertExpectations(t)
}
```

3.  **Race Detection**
    *   Tests must pass with `-race`.
    *   Do not leave dangling goroutines.
