---
name: nido-vm-testing
description: Design or run disposable Nido VM tests, reusable template labs and isolated agent environments through CLI or MCP, with scoped event subscriptions and cleanup of owned resources.
---

# Nido VM Testing

Use Nido for VM-backed tests when the result depends on a guest OS, kernel,
services or networking. Follow the requested test scope; inspect host readiness
with `nido doctor` when needed and select an available image with `nido images list`.

## Disposable test workflow

1. Choose an image or a prepared template, explicit CPU/memory and a unique VM
   name. Check storage before downloads or VM creation; linked clones retain
   dependencies on their base images. Keep existing VMs, templates and caches.
2. Provision prerequisites with `--user-data`. Keep setup non-interactive and
   idempotent, and separate stable template prerequisites from changing code.
3. Spawn with automatic host ports, then read `nido info <name> --json`.
   Its VM object is `data.vm`; discover `ssh_port`, `ssh_user` and labeled
   `forwarding` entries there instead of guessing ports or users.
4. Wait for SSH with a bounded deadline, then check cloud-init and the actual
   service/test readiness. Process-running events do not prove guest readiness.
5. Transfer only the intended test inputs, run the test and export useful logs
   and artifacts before cleanup. Keep secrets out of reusable templates.
6. Delete only the exact resources created by this run, using a trap or CI
   cleanup. Reconcile a failed spawn's partial effects before retrying or deleting;
   preserve unrelated resources and retained debugging evidence.

For a complete shell workflow, inspect or adapt
[scripts/nido-ephemeral-test.sh](scripts/nido-ephemeral-test.sh). It waits separately
for SSH and cloud-init, uploads `NIDO_PROJECT_DIR` and deletes its successful spawn
on exit. `NIDO_KEEP_VM=1` retains the VM on exit, including after test failure.

## Choose the guest architecture

Use omitted/`auto` architecture for native catalog guests. For cross-ISA testing
pass CLI `--arch amd64|arm64` or MCP `arch`; expect TCG rather than KVM/HVF/WHPX.
Do not treat an ARM guest emulated on x86 as native ARM hardware qualification.
Templates inherit their declared ISA and reject explicit conflicts. Verify the
guest with `uname -m` over SSH, separately from QEMU process state. A missing
image variant is an error, not permission to substitute a foreign guest.
For a pre-architecture installation, stop legacy VMs and close old clients before
updating. State conversion is automatic during current updates/installers, or
at the first invocation after an older updater/package manager. Do not run a
manual migration as a routine upgrade step. Preserve disks and backing paths;
follow the [architecture guide](https://github.com/Josepavese/nido/blob/main/docs/architectures.md)
for preparation failures and explicit recovery/adoption of unidentified imports.

## Observe through CLI or MCP

With Nido v4.8.7 or later, use events for shared state changes across independent
CLI, TUI and MCP sessions. TUI synchronization is automatic. Existing clients
must reconnect/reopen to load an upgraded executable; do not restart user VMs
just to activate observation.

- CLI: `nido events --json` emits one response envelope per line; the event is in
  `data`. Cancel and join the owned listener when the workflow ends.
- MCP: read resources for state and use `nido_events` or standard
  `resources/subscribe` for session-local subscriptions. Prefer the owned VM's
  `nido://vm/<name>` URI and relevant outcome kinds over a fleet-wide feed.
- Subscribe before the operation being observed. Notifications are silent by
  default, coalesced and URI-only: re-read the affected resource when notified.
  Do not forward every event into agent conversation or treat notifications as
  tool results. Unsubscribe when observation is no longer needed.
- Events are advisory, with no replay or exactly-once guarantee. Re-read after
  reconnect, resync or an observation gap; inspect `nido_events` status or
  `nido://system/events` for health. Keep bounded readiness/deadline checks so
  missing events cannot stall a job. A failed operation can leave partial state.

Read [references/patterns.md](references/patterns.md) for provisioning/templates,
port routing, guest readiness, subscription examples and failure recovery.
The authoritative [event contract](https://github.com/Josepavese/nido/blob/main/docs/events.md)
and [MCP guide](https://github.com/Josepavese/nido/blob/main/docs/mcp.md) cover the
full catalog and protocol; use the installed server's tool schema for arguments.
