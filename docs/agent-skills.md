# Agent Skills for Nido

Nido ships an optional Codex-compatible skill for developers who want AI agents to use Nido as a VM-backed test and isolation layer.

The skill lives in:

```text
extras/skills/nido-vm-testing/
```

It is distributed for Nido users. It is not part of this repository's local `.codex/` or `.agent/` workspace state.

## Install

From a checkout of this repository:

```bash
mkdir -p "${CODEX_HOME:-$HOME/.codex}/skills"
cp -R extras/skills/nido-vm-testing "${CODEX_HOME:-$HOME/.codex}/skills/"
```

PowerShell:

```powershell
$codexHome = if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $HOME ".codex" }
New-Item -ItemType Directory -Force -Path (Join-Path $codexHome "skills") | Out-Null
Copy-Item -Recurse -Force extras\skills\nido-vm-testing (Join-Path $codexHome "skills")
```

Then restart the agent client so it can discover the new skill.

## What The Skill Teaches

The `nido-vm-testing` skill teaches an agent to:

- use `nido doctor`, `nido images list`, `nido spawn`, `nido info --json`, `nido ssh`, `nido template`, and `nido delete`;
- create disposable VMs for production-like tests;
- provision prerequisites with `--user-data`;
- upload software under test into the VM;
- parse dynamically assigned SSH and service ports;
- run tests without polluting the host;
- convert slow setup into reusable templates;
- isolate multiple AI agents in separate VMs while connecting them through explicit ports;
- observe shared state changes with `nido events` and opt-in MCP subscriptions,
  mute controls and resource re-reads, without treating process events as guest readiness.

Event guidance requires Nido v4.8.7 or later. Updating the executable does not
update a copied skill: refresh `nido-vm-testing` from this repository and restart
the agent client to load its instructions. The skill is distributed separately
from the release binaries.

## Example: Ephemeral VM Test

The bundled script implements the full loop:

```bash
NIDO_TEST_IMAGE=ubuntu:lts \
NIDO_PROVISION_PACKAGES="ca-certificates curl git make build-essential" \
NIDO_TEST_CMD="make test" \
extras/skills/nido-vm-testing/scripts/nido-ephemeral-test.sh
```

Flow:

1. spawn a VM;
2. run first-boot provisioning;
3. discover SSH and forwarded app ports from `nido info --json`;
4. upload the current project;
5. run the test command inside the guest;
6. delete the VM on exit.

Set `NIDO_KEEP_VM=1` to keep the VM for debugging after a failure.

## Example: Template-Accelerated Tests

If provisioning is slow, build a base VM once:

```bash
nido spawn app-base --image ubuntu:lts --user-data provision.sh --memory 4096 --cpus 2
nido ssh app-base "command -v make && command -v git"
nido ssh app-base sudo poweroff
nido info app-base
# After the guest has exited:
nido template create app-base app-ci-base
nido delete app-base
```

Then each test can start from the template:

```bash
nido spawn app-test-001 app-ci-base --port app:8080/tcp
```

This reuses the prepared disk environment. Guest boot and SSH readiness still
take time and depend on the host and image.

## Example: Multi-Agent Isolation

Each AI agent can run in its own VM:

```bash
nido spawn planner --image ubuntu:lts --port api:8000/tcp
nido spawn worker-a --image ubuntu:lts --port api:8000/tcp
nido spawn worker-b --image ubuntu:lts --port api:8000/tcp
```

Use `nido info <vm> --json` to discover each host port. A host-side controller should pass only the required peer endpoints to each agent.

Inside a guest, `127.0.0.1` is the guest itself. For direct guest-to-guest calls through host forwarding, test the QEMU host gateway route, commonly `10.0.2.2:<host_port>`. Prefer a host broker for portable multi-agent labs.

## Port Rules

Nido forwards ports through localhost:

- SSH: host `127.0.0.1:<ssh_port>` to guest `:22`.
- Auto service port: `--port api:8000/tcp`.
- Fixed service port: `--port api:8000:33000/tcp`.
- UDP: `--port dns:53:32053/udp`.

Automatic service ports come from the configured host range, normally `30000-32767`. For a CI worker or a dense multi-agent machine, reserve a dedicated range:

```bash
nido config set PORT_RANGE_START 33000
nido config set PORT_RANGE_END 33999
nido config
```

Best practice: label ports and parse `nido info --json`; do not assume fixed ports in CI or parallel agent runs.
