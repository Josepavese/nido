# [P1] A stale VM PID can make stop terminate an unrelated process

Date: 2026-10-19
Status: fixed in v4.5.29; originally reproduced on Linux v4.5.28 through CLI and MCP.

## Reproduction and impact

Inside a private PID namespace, start a disposable `/usr/bin/sleep 120` and seed
a synthetic `fixture.json` with its PID. This models a stale persisted PID
reused by another process; no real VM metadata was edited.

Both `nido info fixture --json` and `nido_vm action=info` classify it as running.
Both `nido stop fixture --json` and `nido_vm action=stop` then terminate the sleep
process with SIGINT (return code -2), reporting success.

Persisted PIDs survive crashes/reboots and can be reused. Stop, and operations
calling stop such as delete/template creation, can therefore signal an unrelated
process owned by the same user. No actual PID-reuse incident on the user's
machine is asserted; the dangerous behavior is reproduced with a seeded fixture.

## Cause and required outcome

`internal/provider/qemu_process_unix.go:12` only checks signal-0 liveness.
`internal/provider/qemu.go:757` sends signals using the saved numeric PID without
checking process identity or ownership of the VM instance. Its later kill path
has the same issue.

Keep process identity and signalling behind the existing provider/PAL boundary.
Verify the current process belongs to the expected VM instance before reporting
running or signalling it, including PID reuse during the check/signal interval.
Add CLI/MCP tests with an unrelated disposable process, an absent PID, and a
genuine managed QEMU process. Preserve unrelated processes and existing disks.

Evidence: `stale-pid.json`; safe reproducer case `stale-pid`.
See [audit scope and evidence](../../docs/audits/2026-10-19-cli-mcp.md).

## Resolution, v4.5.29

Fixed in the shared provider/configuration boundary. The tracked regression suite
`tests/cli_mcp_safety.py` exercises both CLI and MCP against isolated synthetic
fixtures; the relevant defect now passes, with preservation and positive controls.
See the audit report for the complete validation scope. No existing VM was used
for qualification and no data conversion or VM-state migration is required.
