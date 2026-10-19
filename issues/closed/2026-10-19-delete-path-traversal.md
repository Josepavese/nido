# [P1] VM deletion accepts traversal paths and deletes files outside vms/

Date: 2026-10-19
Status: fixed in v4.5.29; originally reproduced on v4.5.28 through CLI and MCP.

## Reproduction and impact

In an isolated Nido home, create the disposable sentinel
`images/protected-cli.qcow2`, then run:

```sh
nido delete ../images/protected-cli --json
```

The command returns exit 0 and `result=deleted`; the image outside `vms/` is
removed. The equivalent `nido_vm` call with
`{"action":"delete","name":"../images/protected-mcp"}` also removes its sentinel.
Neither request identifies an actual VM. Filesystem permissions still apply;
this is not a claim of privilege escalation. A malformed VM name can nevertheless
delete other writable images or templates, bypassing their normal usage guards.

## Cause and required outcome

`internal/provider/qemu.go:806` joins the unvalidated name into disk and runtime
paths. `filepath.Join` normalizes `..`. The spawn-only validation does not protect
delete, and surface-only checks would leave other consumers exposed.

Validate VM identifiers and path containment in the provider/storage owner
before stop or deletion has any effect. Share that validation across lifecycle
operations; preserve valid names. Check CLI and MCP rejection of traversal,
separators and empty names, unchanged sentinels, and normal deletion of a
disposable valid VM. Do not solve this by weakening backing-file protections.

Evidence: `path-traversal.json`; safe reproducer case `path-traversal`.
See [audit scope and evidence](../../docs/audits/2026-10-19-cli-mcp.md).

## Resolution, v4.5.29

Fixed in the shared provider/configuration boundary. The tracked regression suite
`tests/cli_mcp_safety.py` exercises both CLI and MCP against isolated synthetic
fixtures; the relevant defect now passes, with preservation and positive controls.
See the audit report for the complete validation scope. No existing VM was used
for qualification and no data conversion or VM-state migration is required.
