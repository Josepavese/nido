# I/O Boundaries

Nido separates domain logic from presentation and protocol boundaries.

## Rule

User-facing output and input are allowed only in boundary layers:

- `cmd/*`
- `internal/ui`
- `internal/mcp` for MCP protocol transport only

All other layers must be silent by default.

In particular, these packages must not write directly to user I/O streams:

- `internal/provider`
- `internal/pkg/sysutil`
- `internal/image`
- `internal/builder`

They must not use:

- `fmt.Print*`
- `fmt.Fprint*` with `os.Stdout` or `os.Stderr`
- `println`
- `os.Stdout`
- `os.Stderr`
- `os.Stdin`

Instead they must communicate through:

- return values
- errors
- callbacks
- injected reporters or loggers

## Why

This keeps:

- CLI JSON mode deterministic
- CLI and TUI styling centralized
- provider/PAL/image logic reusable
- protocol boundaries explicit

## Exceptions

- `internal/mcp` may use `stdin/stdout` because it implements the stdio protocol boundary.
- Low-level protocol writes to non-user streams, such as a QMP socket connection, are allowed.
- Tests may capture or replace `os.Stdout` / `os.Stderr`.

## Current enforcement

The repository contains a test that fails if forbidden direct I/O is introduced in the protected packages:

- `internal/architecture/io_policy_test.go`

If you need new runtime feedback from a low-level component, add an injected callback/reporter instead of printing directly.
