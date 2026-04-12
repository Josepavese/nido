# CLI Architecture

Nido now uses a manifest-driven CLI instead of a hand-written command switch.

## Single Source of Truth

The command tree lives in:

- `internal/cli/commands.yaml`

That file defines:

- command ids
- `use`, aliases, groups, descriptions, examples
- flags attached to each command
- completion sources
- action ids

Help output and shell completion are generated from the same command tree, so adding or changing a command no longer requires editing multiple hardcoded paths.

## Runtime

The runtime is built in two layers:

- `internal/cli/manifest.go`
  - loads and validates the manifest
- `internal/cli/builder.go`
  - turns the manifest into a Cobra command tree

The `cmd/nido` package only provides two things:

- action handlers
- completion providers

The entrypoint in `cmd/nido/main.go` only bootstraps the app context and executes the generated Cobra tree.

## Command Wiring

Handlers are registered by stable action ids:

- `cmd/nido/actions_registry.go`

This keeps the manifest decoupled from Go symbol names. The manifest refers to ids like `vm.spawn` or `system.update`, and Go maps those ids to concrete handler functions.

## Completion Wiring

Completion sources are registered in:

- `cmd/nido/completions.go`

The manifest references completion ids, not shell scripts. Cobra generates Bash/Zsh/Fish/PowerShell completion from the same runtime tree.

## Adding a Command

1. Add or update the command in `internal/cli/commands.yaml`.
2. Add the handler in `cmd/nido/actions_*.go`.
3. Register the handler id in `cmd/nido/actions_registry.go`.
4. If needed, add a completion source in `cmd/nido/completions.go`.
5. Run:
   - `go test ./...`
   - `go vet ./...`

## Rules

- Do not add manual help text in `main.go`.
- Do not add handwritten shell completion scripts.
- Do not dispatch commands with manual `switch` logic.
- If a command needs metadata, put it in the manifest first.

The CLI should stay declarative at the metadata layer and explicit at the handler layer.
