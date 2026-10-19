# [P2] Info reports success for nonexistent VMs and hides corrupt runtime state

Date: 2026-10-19
Status: fixed in v4.5.29; originally reproduced on v4.5.28 through CLI and MCP.

## Reproduction and impact

With no disk or state for `does-not-exist`:

```sh
nido info does-not-exist --json
```

Exit code is 0, status is `ok`, and a stopped VM with zero resources is returned.
The CLI JSON omits `DiskMissing`, so automation cannot distinguish this response
from a real stopped VM using that field. MCP `nido_vm action=info` also reports
success, although its detail contains `DiskMissing=true`.

A second synthetic fixture has a disk but malformed `run/corrupt.json`. Both
surfaces report success and return zero resources instead of exposing the parse
error. This can turn missing/corrupt state into a false readiness signal.

## Cause and required outcome

`internal/provider/qemu.go:662` treats every `loadState` failure as a stopped VM,
including invalid JSON and the simultaneous absence of disk and state.
`cmd/nido/actions_vm.go:93` projects only part of disk-health information into
CLI JSON, despite having an existing not-found error branch.

Distinguish not-found, malformed/unreadable state and an explicitly supported
recoverable disk-only state in the provider contract. Propagate actionable errors
or explicit degraded health consistently to CLI/MCP; do not rewrite or reset
useful state. Test missing VM, corrupt state, healthy stopped VM and orphan disk.

Evidence: `missing-info.json`; safe reproducer case `missing-info`.
See [audit scope and evidence](../../docs/audits/2026-10-19-cli-mcp.md).

## Resolution, v4.5.29

Fixed in the shared provider/configuration boundary. The tracked regression suite
`tests/cli_mcp_safety.py` exercises both CLI and MCP against isolated synthetic
fixtures; the relevant defect now passes, with preservation and positive controls.
See the audit report for the complete validation scope. No existing VM was used
for qualification and no data conversion or VM-state migration is required.
