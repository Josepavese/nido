---
name: nido-vm-testing
description: Use when designing, writing, or running Nido-backed VM tests, disposable production-like QA environments, template-accelerated test labs, or isolated multi-agent AI sandboxes. Provides workflows for Nido CLI spawn/provision/upload/test/delete automation, port forwarding, templates, cleanup hygiene, and VM isolation best practices.
---

# Nido VM Testing

Use Nido as a disposable VM lab for tests and agent workloads. Prefer real VMs when the result depends on system packages, init behavior, kernel isolation, networking, filesystem layout, OS services, or production-like assumptions that containers do not faithfully model.

## Quick Start

Run the host preflight first:

```bash
nido doctor
nido images list
```

Minimal lifecycle:

```bash
cat > provision.sh <<'EOF'
#!/bin/sh
set -eux
apt-get update
apt-get install -y ca-certificates curl git make
EOF

nido spawn app-test --image ubuntu:24.04 --user-data provision.sh --port app:8080/tcp --memory 4096 --cpus 2 --json
nido info app-test --json
nido ssh app-test "echo ready"
nido delete app-test --json
```

For a ready automation example, run or adapt [scripts/nido-ephemeral-test.sh](scripts/nido-ephemeral-test.sh).

## Standard Workflow

1. Choose an image or template.
   Use cloud images for clean OS tests. Use templates when prerequisites are expensive and stable.
2. Write provisioning as a `--user-data` script.
   Install OS packages, users, services, certificates, language runtimes, and test dependencies here.
3. Spawn with explicit resources and port rules.
   Use `--memory`, `--cpus`, and `--port label:guest/tcp`; let Nido allocate host ports unless a fixed port is required.
4. Wait for SSH using `nido info --json`.
   Parse `ssh_port` and `ssh_user`; do not assume fixed ports.
5. Upload or fetch the software under test.
   Use direct `ssh`/`scp` with the port from `nido info`, or clone from an internal repository.
6. Run tests inside the VM.
   Keep commands non-interactive and capture logs/artifacts before cleanup.
7. Delete the VM.
   Use shell traps or CI cleanup steps. Avoid leaving stopped VMs unless debugging.

## Best Practices

- Use unique names such as `nido-ci-$RUN_ID` and delete by exact name.
- Always parse `nido info --json`; ports are runtime allocations.
- Prefer auto host ports: `--port api:8000/tcp`. Fixed host ports are for human workflows, not parallel CI.
- Keep provisioning idempotent. A script should be safe if cloud-init retries it.
- Put slow, stable prerequisites into a template. Put changing app code into the ephemeral VM at test time.
- Pin OS images, package versions, and test commands for reproducibility.
- Keep secrets out of templates. Inject short-lived secrets at runtime and scrub logs.
- Use `nido delete <name>` for test cleanup. Avoid broad `nido prune` on shared developer machines.
- For Windows blueprints, use `--gui` on the first boot when diagnosing OOBE/post-install state.

## Ports

Nido uses localhost forwarding. Each VM gets an SSH host port and any custom forwarded service ports.

Examples:

```bash
nido spawn api-1 --image ubuntu:24.04 --port api:8000/tcp
nido spawn web-1 --image ubuntu:24.04 --port web:3000:33000/tcp
nido info api-1 --json
```

Port forms:

- `8000` forwards an auto-selected host port to guest `8000/tcp`.
- `api:8000/tcp` adds label `api` and auto-selects the host port.
- `api:8000:33000/tcp` maps host `127.0.0.1:33000` to guest `8000`.
- `dns:53:32053/udp` maps UDP.

Nido allocates automatic service ports from its configured range, normally `30000-32767`. For dense CI or multi-agent hosts, reserve a dedicated range:

```bash
nido config set PORT_RANGE_START 33000
nido config set PORT_RANGE_END 33999
nido config
```

Treat host ports as the public contract between the controller and VMs. For VM-to-VM agent communication, prefer a host-side broker/controller that knows each forwarded port. If a guest must call another guest through host forwarding, inject the discovered peer endpoint and verify whether the guest can reach the host gateway, commonly `10.0.2.2:<host_port>` under QEMU user networking.

## Use Cases

### Ephemeral Production-Like Tests

Create a clean VM, install prerequisites through cloud-init, upload the current project, run the tests, collect output, then delete the VM. This avoids polluting the developer host and catches OS-level integration bugs.

Use the bundled script:

```bash
NIDO_TEST_IMAGE=ubuntu:24.04 \
NIDO_PROVISION_PACKAGES="ca-certificates curl git make build-essential" \
NIDO_TEST_CMD="make test" \
extras/skills/nido-vm-testing/scripts/nido-ephemeral-test.sh
```

### Template-Accelerated Tests

When provisioning takes minutes, build a base VM once and save it as a template:

```bash
nido spawn app-base --image ubuntu:24.04 --user-data provision.sh --memory 4096 --cpus 2
# wait for provisioning, verify prerequisites
nido stop app-base
nido template create app-base app-ci-base
nido delete app-base

nido spawn app-test-001 app-ci-base --port app:8080/tcp
```

Template spawns are fast, typically milliseconds on a warm cache, because each VM is a linked clone. Keep the template generic; copy the changing code into each test VM.

### Multi-Agent AI Labs

Run each agent in its own VM and connect them through explicit ports:

```bash
nido spawn agent-a --image ubuntu:24.04 --port api:8000/tcp
nido spawn agent-b --image ubuntu:24.04 --port api:8000/tcp
nido info agent-a --json
nido info agent-b --json
```

The host/controller should parse both `info` payloads, assign peer URLs, and start each agent with only the ports it needs. This preserves filesystem/process isolation while still allowing controlled communication.

## References

Read [references/patterns.md](references/patterns.md) for more complete workflow notes, CI guidance, and multi-agent port patterns.
