# Hydra Pattern: Interface-First Design

**Rule:** NEVER define a concrete struct without an interface.
**Reason:** Enables total testability via mocks.

## The 3-Step Pattern

You must follow this exact sequence for every component:

### 1. Interface (The Contract) - Exported
Define *what* it does.
```go
// Manager controls the process lifecycle.
type Manager interface {
    Start() error
    Stop() error
}
```

### 2. Struct (The Implementation) - Unexported
Define *how* it works. Keep fields private.
```go
type manager struct {
    cmd    *exec.Cmd
    logger logger.Logger
}

func (m *manager) Start() error { ... }
```

### 3. Constructor (The Factory) - Exported
Inject dependencies. Return the **Interface**, not the struct.
```go
// NewManager creates a standard process manager.
func NewManager(l logger.Logger) Manager {
    return &manager{
        logger: l,
    }
}
```

## Anti-Patterns (FORBIDDEN)
❌ `type Manager struct { ... }` (Public struct)
❌ `func New() *Manager` (Returning concrete type)
❌ Global state or `init()` functions.
