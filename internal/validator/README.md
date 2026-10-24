# Validation engine

The [validator guide](../../cmd/nido-validator/README.md) owns command-line usage,
environment settings, outputs and cleanup behavior. The
[testing guide](../../TESTING.md) explains isolation and qualification scope.

Shared end-to-end workflows live in `workflows/default.yaml` and are executed
through CLI and MCP. Add connected scenarios in `scenario/` and wire them through
`cmd/nido-validator/main.go`. Keep both public surfaces covered; respect generated
resource ownership and preserve failed recovery evidence.

Build and log outside the checkout. Reuse the candidate binary and test cache for
related consumers, then remove owned temporary artifacts after writers exit.
