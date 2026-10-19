# [P1] MCP image removal bypasses the backing-file guard used by CLI

Date: 2026-10-19
Status: fixed in v4.5.29; originally reproduced on v4.5.28 with real synthetic qcow2 overlays.

## Reproduction and impact

Create a 1 MiB virtual base `images/audit-v1.qcow2` and an overlay
`vms/fixture.qcow2` with that absolute backing path. Verify the chain with
`qemu-img info --backing-chain --output=json`.

`nido cache remove audit:v1 --json` correctly exits 1: `in use by a VM`.
On the same fixture, MCP `nido_image` with
`{"action":"cache_remove","image":"audit:v1"}` reports `status=removed`, deletes
the base and makes the overlay's backing-chain inspection fail.

The same discrepancy is independently reproduced for CLI `images remove` versus
MCP `action=remove`. The VM need not be running to depend on its base image.
Deleting that base makes the dependent VM disk unusable.

## Cause and required outcome

`internal/mcp/mcp.go:640` and `:681` call
`image.Catalog.RemoveCachedImage`, which directly removes the file
(`internal/image/catalog.go:216`). CLI uses `VMProvider.CacheRemove`, including
the usage guard. This is duplicated destructive behavior across owners.

Route both MCP removal actions through the canonical storage/provider contract.
Test both CLI commands and both MCP actions against the same referenced fixture:
all must refuse and preserve the chain. Also verify removal of an unreferenced
image. Keep this distinct from the separate incomplete backing-discovery issue.

Evidence: `cache-direct.json`; safe reproducer case `cache-direct`.
See [audit scope and evidence](../../docs/audits/2026-10-19-cli-mcp.md).

## Resolution, v4.5.29

Fixed in the shared provider/configuration boundary. The tracked regression suite
`tests/cli_mcp_safety.py` exercises both CLI and MCP against isolated synthetic
fixtures; the relevant defect now passes, with preservation and positive controls.
See the audit report for the complete validation scope. No existing VM was used
for qualification and no data conversion or VM-state migration is required.
