# [P2] Configured automatic port forwarding is silently omitted at restart

Date: 2026-10-19
Status: fixed in v4.5.29; originally reproduced on v4.5.28 through CLI and MCP, with native QEMU.

## Reproduction and impact

For a disposable stopped fixture:

```sh
nido config fixture --port=http:80 --json
nido start fixture --json
```

Both succeed. Persistent forwarding contains `guest_port=80, host_port=0`.
The actual QEMU process has only
`-netdev user,id=net0,hostfwd=tcp:127.0.0.1:51022-:22`: port 80 is absent.

MCP `nido_vm config_update` with `ports=["http:80"]`, followed by `action=start`,
has the same result. An explicit `http:80:31080` works on both surfaces and
appears in the native process arguments after restart.

Tests used a tiny empty qcow2 disk, 128 MiB memory and QEMU TCG paused with `-S`.
This proves omission from the real runtime arguments; it is not a guest HTTP
service qualification. No pre-existing VM was started or changed.

## Cause and required outcome

`internal/provider/qemu.go:1069` saves accepted zero-host-port mappings unchanged.
Spawn and `PortForward` allocate automatic host ports, but `Start` does not.
`BuildNetDevArgs` (`:1103`) silently skips rules with `HostPort=0`.

Use one canonical allocation/resolution path for saved forwarding before launch,
persist resolved values, and report allocation failures. Cover CLI and MCP config
updates, omission, restart, multiple mappings, explicit ports and exhausted
ranges. Do not add surface-specific allocation logic.

Evidence: `config-port.json`; safe reproducer case `config-port`.
See [audit scope and evidence](../../docs/audits/2026-10-19-cli-mcp.md).

## Resolution, v4.5.29

Fixed in the shared provider/configuration boundary. The tracked regression suite
`tests/cli_mcp_safety.py` exercises both CLI and MCP against isolated synthetic
fixtures; the relevant defect now passes, with preservation and positive controls.
See the audit report for the complete validation scope. No existing VM was used
for qualification and no data conversion or VM-state migration is required.
