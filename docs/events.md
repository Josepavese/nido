# Shared local events

This page owns the event catalog, delivery semantics and qualification checkpoint.

Nido state files and provider queries remain authoritative. Events tell a
consumer what changed; they do not execute commands or become a second VM DB.

## Event analysis

| Family | Events worth exposing | Meaning and consumers |
| --- | --- | --- |
| Nido operations | started, completed, failed for VM create/start/stop/delete/configure/port changes/disk creation, templates, cache, catalog refresh, global config, image acquisition, blueprint builds, self-update and uninstall | A correlated intent and terminal API outcome. Failure can follow partial effects. Completion does not imply guest OS/SSH/application readiness. TUI refreshes at terminal outcome; MCP omits starts. |
| Nido acquisition/configuration | image acquisition, blueprint build, global configuration changes | Terminal core resource changes, without byte progress, URLs, scripts, credentials or configuration values. |
| Runtime observations | process running/stopped; QMP pause, resume, reset, shutdown, suspend, wakeup, guest panic, watchdog and block I/O error | Enrich lifecycle state. QMP SHUTDOWN means shutdown announced, not process termination. A VM disk disappearing is a resource removal, never proof that its process stopped. Only verified process identity establishes stopped. No guest boot/readiness claim. |
| Persistent resource invalidation | VM metadata/PID, disk create/remove/rename, template publication/removal, image/catalog/config publication | External or other-process changes invalidate the same resources. Native watcher backend is confined to PAL. Disk content writes, staging/partial download writes and unrelated files are excluded. |
| Stream control | resync, unavailable | The consumer must refresh its projection or show degraded connectivity. Reconnect/new owner has a new epoch. Missing observations are never a successful state. |

Excluded: SSH output, guest logs/telemetry, terminal transcripts, tool arguments,
passwords, QMP raw payloads, display/network activity and high-frequency progress.
Read-only calls are not events. PCI operations remain under the existing HAL;
this feature neither detaches hardware nor invents hardware success events.

## Transport and lifetime

A TUI or MCP session connects to `events/bus.sock` under its Nido root. If no
broker is reachable, an OS lock elects one of these sessions as owner. All
sessions attach to that owner, which alone runs filesystem/runtime observers.
On owner exit another session takes over. The last session releases observers,
connections and the socket; no resident child daemon, system service or journal
is installed. `owner.lock` is an empty reusable lock inode, not a lease record.

PAL uses AF_UNIX on Linux/macOS and Windows 10 1803 or later. The Unix directory
is private (0700), socket 0600. On Windows it inherits the user's home ACL. The
bus does not bind a network TCP port. A stale endpoint may be removed only by
the lock owner, and a non-socket endpoint is never overwritten.

The local protocol uses HTTP over the socket: `POST /publish` accepts validated
Nido operation events; `GET /watch` returns NDJSON. Runtime/filesystem producers
are internal to the broker. Version, broker epoch and monotonic sequence order
published observations. An operation ID correlates start/outcome for one operation. Nested and concurrent
operations have distinct IDs. API outcomes can include partial effects; re-read
the affected resources after a failure. Uninstall can remove the bus endpoint
before its terminal event, so its direct tool/CLI result remains decisive.

New subscribers receive the latest observed runtime detail after the resync
barrier. These frames carry `snapshot=true` and the current sequence watermark;
their timestamp is the original observation. The cache is subordinate to the
current observer epoch and discarded on observer unavailability, process stop
or inventory removal. A resync re-emits the still-known observations.

The stream is advisory, in memory, without history replay or exactly-once
claims. A new connection always starts with `stream.resync`; re-read resources.
Slow subscriber queues are bounded and disconnect on overflow, then reconnect
and resync. A stopped observer cannot block VM operations indefinitely. Limits
and timings belong to `internal/events`, with MCP batching owned by
`internal/mcp/events.go`.

The broker uses native directory notifications and native process-exit waits
(pidfd on Linux, kqueue on macOS, process handles on Windows). It validates
process identity again before publishing stopped. A central reconciliation
interval also discovers runtimes and recovers from unavailable native waits,
watcher loss and existing VMs without the dedicated event monitor. There is no per-TUI
fleet polling timer. Existing running VMs gain their dedicated QMP event channel
on their next ordinary start; no user VM is restarted for this feature.

QMP observation uses a second monitor, preserving the existing control monitor.
Unix uses a socket under the private events directory; Windows QEMU uses an
ephemeral loopback TCP monitor whose address is persisted in that directory. QMP greeting and
VM name are checked before event delivery. Initial `prelaunch` (QEMU `-S`) is
reported as paused CPU execution, without implying guest readiness. Only selected event names leave the
adapter; raw guest/device details do not.

## MCP: opt in and low noise

The server advertises resource subscriptions. Standard `resources/subscribe`
and `resources/unsubscribe` control a resource. `nido_events` exposes the same
session controls to agents that only call tools: `status`, `subscribe`,
`unsubscribe`, `mute`, `unmute`. URI scopes include fleet, one VM, templates,
images, blueprints, cache, config, or `nido://system/events`. Optional kind filters
apply to tool subscriptions. Unmute invalidates the resource once to recover
suppressed changes. Subscriptions are session local and disappear at EOF.

Notifications use standard `notifications/resources/updated` and contain only
`uri`. The client decides when to read it. None are emitted without an explicit
subscription, or before the initialized notification. Multiple observations of
the same resource are coalesced. The event resource contains session health,
subscriptions and a bounded window of interested events; it is not an audit log.
The `resync_required` flag records a gap during the session. Resource notifications
are advisory invalidations, not a guarantee that the client has refreshed.

For terminal observation use `nido events` or `nido events --json` (one standard
CLI response envelope per line). Shell completion discovers the command from the
canonical command manifest. Ctrl+C releases the observer owned by this command.

Examples:

```json
{"method":"resources/subscribe","params":{"uri":"nido://vm/build-box"}}
{"method":"tools/call","params":{"name":"nido_events","arguments":{"action":"subscribe","uri":"nido://vm/build-box","kinds":["vm.stopped","vm.panic","operation.failed"]}}}
{"method":"tools/call","params":{"name":"nido_events","arguments":{"action":"mute","uri":"nido://vm/build-box"}}}
```

## Qualification checkpoint

- Baseline: clean `main` at `8d8942f`. Event contract, native PAL transport,
  election, filesystem/process waits, domain producers, QMP, TUI and MCP are implemented.
- Verified: full Go tests and vet; focused race tests; Linux real QEMU
  pause/resume/stop with a private 1 MiB disk and 128 MiB RAM; validator event-stream
  scenario; Windows amd64 and macOS arm64 application compilation.
- Verified integration: two actual Unix PTYs, hidden-route refresh, CLI-to-MCP
  propagation, default silence, mute/unmute, external atomic updates/removal and
  broker crash takeover. Final changed-boundary tests, race detector, vet and
  isolated QEMU qualification all pass.
- Remaining: native Windows/macOS CI and repository/wiki publication.
- The initial QEMU attempt missed `prelaunch`; the corrected runtime mapping
  passed. PTY observer assertions were corrected against the actual MCP `vm`
  envelope, nullable empty fleet and rendered route text; these were harness
  corrections, not proofs of the final path before the corrected run.
- Scratch owner: this event goal. `/tmp/nido-events-build` is the shared external
  build/test area; limit 2 GiB; retire after final consumer. Shared Go module and
  build caches are reused, not removed. No existing VM or user data is modified.

## Protocol references

- [MCP resource subscriptions](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2025-06-18/schema.mdx): standard subscribe/unsubscribe and URI-only update notifications.
- [QEMU QMP reference](https://www.qemu.org/docs/master/interop/qemu-qmp-ref.html): lifecycle events, run states and SHUTDOWN limitations.
- [fsnotify](https://github.com/fsnotify/fsnotify): native filesystem backends, hidden behind Nido PAL.

## Public vocabulary and resource routing

The named constants and validation in `internal/events/events.go` own this
vocabulary. Each operation uses `operation.started` and exactly one local
terminal outcome, `operation.completed` or `operation.failed`, correlated by
`operation_id`. Delivery of either frame is advisory.

| Actions | Invalidated resources |
| --- | --- |
| `vm.create`, `vm.start`, `vm.stop`, `vm.delete`, `vm.configure`, `vm.port_forward`, `vm.port_unforward`, `disk.create` | Fleet and named VM |
| `template.create`, `template.delete` | Template fleet |
| `image.acquire`, `blueprint.build`, `cache.remove`, `cache.prune` | Cache, image and blueprint catalogs |
| `catalog.refresh` | Image and blueprint catalogs |
| `config.set` | Global configuration |
| `system.update` | Version and catalogs; reconnect to execute the new binary |
| `system.uninstall` | Core resources; removing the endpoint may end delivery before its terminal event |

| Observation kinds | Interpretation |
| --- | --- |
| `vm.running`, `vm.stopped` | Owned process observed alive or confirmed absent; running does not attest guest readiness |
| `vm.paused`, `vm.resumed` | QMP CPU execution paused/prelaunch or resumed |
| `vm.reset` | Reset observed, not completion of guest reboot |
| `vm.shutdown` | QEMU announced shutdown; wait for `vm.stopped` before claiming host-process termination |
| `vm.suspended`, `vm.wakeup` | Guest suspend/hibernate or wakeup observed |
| `vm.panic`, `vm.watchdog`, `vm.io_error` | Core runtime failures requiring inspection; raw payloads are omitted |
| `resources.changed` | Persistent metadata/catalog/config or disk publication/removal invalidates projections |
| `vm.removed` | A previously known VM disk disappeared from the inventory; its process may still exist |
| `stream.resync`, `stream.unavailable` | Refresh projections, or treat observation as unavailable; neither changes VM state |

A broker can be connected while its observer is unavailable. MCP exposes both
`connected` and `observer_available`; permission/backend errors on watcher setup
are visible rather than interpreted as an empty fleet. Missing optional storage
directories are watched when they appear, followed by resync. Native process
wait failures use the documented central reconciliation recovery path.
