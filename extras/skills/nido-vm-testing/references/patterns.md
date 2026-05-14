# Nido VM Testing Patterns

Use this reference when the task needs more detail than the core skill body.

## Ephemeral Test VM

Goal: create a faithful OS test environment without mutating the developer host.

Pattern:

1. Create a user-data provisioning script.
2. Spawn a VM from a pinned image.
3. Parse `nido info --json` for SSH and service ports.
4. Copy the project into `/tmp` or `/opt`.
5. Run tests.
6. Copy artifacts out if needed.
7. Delete the VM in a trap.

Use auto host ports to keep CI parallel:

```bash
nido spawn ci-$RUN_ID --image ubuntu:24.04 --port app:8080/tcp --json
nido info ci-$RUN_ID --json
```

Do not assume host port `50022` for SSH or a fixed application port. Nido chooses free ports and reports them.

## Provisioning

Prefer `--user-data` for first-boot setup:

```bash
cat > provision.sh <<'EOF'
#!/bin/sh
set -eux
apt-get update
apt-get install -y ca-certificates curl git make build-essential
EOF

nido spawn ci-base --image ubuntu:24.04 --user-data provision.sh
```

For cloud images, Nido adds its SSH key and combines the user-data with the base cloud-init payload. Keep provisioning non-interactive.

## Template Acceleration

Use templates when test prerequisites are slow and stable. Once the template exists, linked-clone test VMs usually spawn in milliseconds on a warm cache:

```bash
nido spawn app-base --image ubuntu:24.04 --user-data provision.sh --memory 4096 --cpus 2
nido ssh app-base "command -v node && command -v npm"
nido stop app-base
nido template create app-base app-ci-node20
nido delete app-base

nido spawn app-test-$RUN_ID app-ci-node20 --port app:3000/tcp
```

Keep templates free of:

- application source trees that change often;
- long-lived credentials;
- stale package caches if disk size matters;
- environment-specific hostnames or tokens.

Copy the current software into each linked clone after spawn.

## Port Design

Nido port mappings are host-facing contracts:

- SSH is always host `127.0.0.1:<ssh_port>` to guest `:22`.
- `--port api:8000/tcp` exposes guest `:8000` on an auto-selected host port.
- `--port api:8000:33000/tcp` exposes guest `:8000` on fixed host `:33000`.

Read ports from:

```bash
nido info vm-name --json
```

Configure the automatic host port range when the default `30000-32767` range overlaps with other services or when CI workers need predictable allocation zones:

```bash
nido config set PORT_RANGE_START 33000
nido config set PORT_RANGE_END 33999
nido config
```

Use labels (`api`, `web`, `db`, `metrics`) so orchestration scripts can find the right forwarded port without relying on list order.

## Multi-Agent Labs

For multiple AI agents:

1. Spawn one VM per agent.
2. Give each VM only the service ports it needs.
3. Parse all `nido info --json` payloads.
4. Generate a small peer config for each agent.
5. Start agents through SSH.
6. Keep a host-side controller or broker as the source of truth for peer addresses.

Example:

```bash
nido spawn planner --image ubuntu:24.04 --port api:8000/tcp
nido spawn worker-a --image ubuntu:24.04 --port api:8000/tcp
nido spawn worker-b --image ubuntu:24.04 --port api:8000/tcp
```

From the host, call each service through `127.0.0.1:<host_port>`.

From inside a guest, `127.0.0.1` is the guest itself. If a guest must call another VM through host forwarding, inject the discovered peer endpoint and test the host gateway route, commonly `10.0.2.2:<host_port>` under QEMU user networking. Prefer a host broker when portability matters.

## Cleanup Rules

Use exact VM names:

```bash
nido delete ci-$RUN_ID --json
```

Avoid `nido prune` in shared environments unless the user explicitly wants every stopped VM removed. For CI, delete the VMs the job created.

## Failure Handling

If a test fails:

- keep the VM with `NIDO_KEEP_VM=1`;
- run `nido info <name> --json`;
- connect with `nido ssh <name>`;
- collect logs from `/var/log/cloud-init.log`, `/var/log/cloud-init-output.log`, systemd journals, app logs, and test artifacts;
- delete the VM after diagnosis.
