# Nido VM Testing Patterns

Read the sections relevant to the current test. Values below are examples;
select images, resources and commands for the actual workload.

## Provisioning and readiness

```bash
cat > provision.sh <<'SETUP'
#!/bin/sh
set -eu
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y ca-certificates curl git make
SETUP

nido spawn "ci-$RUN_ID" --image ubuntu:24.04 --user-data provision.sh \
  --memory 4096 --cpus 2 --port app:8080/tcp --json
nido info "ci-$RUN_ID" --json
```

For cloud images, Nido supplies its SSH key alongside the user-data. Discover the
SSH user/port from `data.vm`, wait for SSH with a deadline, then check
`sudo cloud-init status --wait` with a bounded timeout and verify the service or
command the test actually needs. A successful spawn or `vm.running` describes
host/process progress, not completed provisioning. Collect guest logs such as
`/var/log/cloud-init-output.log`, service journals and test results before deleting
the owned VM.

The bundled helper accepts `NIDO_USER_DATA`, `NIDO_PROJECT_DIR`,
`NIDO_REMOTE_DIR`, `NIDO_TEST_CMD`, resource overrides and `NIDO_KEEP_VM`;
run it with `--help` for its current defaults. Its default provisioning uses
Debian/Ubuntu packages. Adapt it for other guests; choose a disposable remote
project directory and a local payload without unintended secrets or large caches.

## Reusable templates

Prepare and verify stable prerequisites once, then shut down the guest cleanly:

```bash
nido spawn app-base --image ubuntu:24.04 --user-data provision.sh
# After SSH, cloud-init and prerequisite checks pass:
nido ssh app-base sudo poweroff
nido info app-base --json
# After confirming the managed process is stopped:
nido template create app-base app-ci-base
nido delete app-base

nido spawn "app-test-$RUN_ID" app-ci-base --port app:8080/tcp
```

Template commands are explicit mutations: use names owned by the workflow and
retain the template while clones depend on it. Reusing prepared storage saves
setup work; guest boot and readiness still take time. Do not preserve changing
source trees, environment-specific configuration or long-lived credentials in a
shared base. Upload current inputs after spawning each clone.

## Ports and multiple agents

Nido forwards host loopback ports into the guest:

- `--port api:8000/tcp`: automatic host port for guest TCP 8000.
- `--port api:8000:33000/tcp`: fixed host port 33000.
- `--port dns:53:32053/udp`: UDP forwarding.

Prefer automatic ports for parallel jobs. Read labeled `data.vm.forwarding`
entries and `ssh_port` from `info`; never infer a port from list position.
Only change the global port range when the task calls for host configuration.

For multi-agent labs, create one VM per required agent and pass only the peer
endpoints it needs. The controller owns the discovered host URLs. Inside a guest,
`127.0.0.1` is the guest itself: host loopback forwarding is not automatically
reachable by another VM. Use a host broker or verify a guest-reachable host
endpoint before advertising it; do not assume a gateway address works.

## Low-noise event observation

These are MCP request bodies; the client supplies normal JSON-RPC framing and IDs:

```json
{"method":"tools/call","params":{"name":"nido_events","arguments":{"action":"subscribe","uri":"nido://vm/build-box","kinds":["operation.failed","vm.stopped","vm.panic"]}}}
{"method":"tools/call","params":{"name":"nido_events","arguments":{"action":"mute","uri":"nido://vm/build-box"}}}
{"method":"tools/call","params":{"name":"nido_events","arguments":{"action":"unmute","uri":"nido://vm/build-box"}}}
{"method":"resources/read","params":{"uri":"nido://vm/build-box"}}
{"method":"tools/call","params":{"name":"nido_events","arguments":{"action":"unsubscribe","uri":"nido://vm/build-box"}}}
```

Choose filters for the operation: this example observes failures and shutdown;
it deliberately does not report successful start completion. Add the needed kinds
or omit the filter for all core changes on that VM. Standard
`resources/subscribe` and `resources/unsubscribe` take the same URI, without kind
filters. Subscribe and read the initial state before performing the operation.

Mute retains filters and suppresses notifications; unmute invalidates once so the
client can recover missed changes. Subscriptions vanish when the MCP session
closes. Read health with `nido_events` action `status`; `connected` and
`observer_available` are distinct, and `resync_required` records a session gap.
A notification means re-read, not that the projected state already refreshed.

For CLI controllers, `nido events --json` starts with `stream.resync`; re-read the
resources of interest. Runtime snapshot frames may describe an earlier
observation. QMP shutdown is not process exit; confirm stopped state before disk
template or cleanup work. Running VMs gain the additional QMP monitor at their
next ordinary start, while filesystem/process observation works immediately.

## Failure recovery and cleanup

On failure, inspect `nido info <name> --json` and the direct CLI/MCP result,
export evidence and reconcile partial effects before retrying. Use
`NIDO_KEEP_VM=1` with the helper if the VM must survive for diagnosis. Record the
retained resource and its next consumer; delete it by exact name after diagnosis.
Avoid broad prune/cleanup on shared hosts. A missing advisory event or an empty
recent-event window proves neither success nor absence of an operation.
