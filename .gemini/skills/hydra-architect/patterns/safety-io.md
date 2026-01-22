# Hydra Pattern: Safety & IO Strictness

**Rule:** `stdout` is for JSON-RPC ONLY. The `fmt` package is BANNED in production.
**Reason:** One rogue print corrupts the connection and kills the AI session.

## IO Rules

1.  **NO `fmt.Println` / `print` / `println`**
    *   Use `internal/logger` for *everything*.
    *   `logger.Info("started")` -> writes to `stderr`.
    *   `fmt.Errorf` is ALLOWED (it returns a string, doesn't print).

2.  **Zombie Prevention**
    *   NEVER spawn a process without `SysProcAttr`.
    *   Must use `Setpgid: true` (Unix) or Job Objects (Windows via library).

3.  **Panic Recovery**
    *   Every goroutine must have a `defer recover()` block if it runs long-term.
    *   Log panics to `stderr` and continue if possible.

## Sanitizer Usage

When reading from a child process:
```go
// GOOD
chunk := read()
if sanitizer.IsJSON(chunk) {
    forward(chunk)
} else {
    logger.Warn("Child spewed garbage", "data", chunk)
}
```
