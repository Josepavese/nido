# [P1] Unused-cache pruning deletes transitive or relative backing images

Date: 2026-10-19
Status: fixed in v4.5.29; originally reproduced on v4.5.28 through CLI and MCP.

## Reproduction and impact

Two independent fixtures use actual qcow2 files, each with 1 MiB virtual size:

1. `vms/fixture.qcow2` backs onto `backups/mid.compact.qcow2`, which backs onto
   `images/base.qcow2`; all backing paths are absolute.
2. `vms/fixture.qcow2` directly backs onto `../images/base.qcow2`, relative to the
   overlay's directory.

Each chain is valid before the operation. Both `nido cache prune --json` and
MCP `nido_image {"action":"cache_prune","unused_only":true}` report success and
delete the required base. `qemu-img info --backing-chain --output=json` fails
afterward. CLI explicitly reports `unused_only=true` while removing that base.

Positive control: the CLI preserves a direct, absolute backing image. The defect
depends on dependency discovery, rather than pruning deleting every used file.

## Cause and required outcome

`internal/provider/qemu.go:1262` records only each VM disk's immediate backing.
It does not traverse ancestors. `backingInfo` (`:730`) returns `backing-filename`;
the usage scan then resolves relative paths against process cwd (`:1284`), rather
than against the containing overlay. The referenced base is absent from the
pruning protection set (`:1482`). Template deletion uses the same discovery.

Make backing-chain discovery canonical in the storage owner, resolve each path
against its image, traverse all ancestors and handle inspection failures
conservatively before destructive operations. Test absolute, relative and
multilevel chains through CLI/MCP, with unreferenced-image removal as a control.
Preserve useful images; do not flatten or copy existing VM disks as a workaround.

Evidence: `cache-chain.json`, `cache-relative.json`, `controls.json`.
Safe reproducer cases: `cache-chain`, `cache-relative`, `controls`.
See [audit scope and evidence](../../docs/audits/2026-10-19-cli-mcp.md).

## Resolution, v4.5.29

Fixed in the shared provider/configuration boundary. The tracked regression suite
`tests/cli_mcp_safety.py` exercises both CLI and MCP against isolated synthetic
fixtures; the relevant defect now passes, with preservation and positive controls.
See the audit report for the complete validation scope. No existing VM was used
for qualification and no data conversion or VM-state migration is required.
