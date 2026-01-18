Nido Validation Suite
=====================

Overview
- `nido-validator` exercises Nido CLI and MCP flows end-to-end: version/doctor, images/cache/templates, VM lifecycle (spawn/info/list/ssh/start/stop/delete/prune), template workflows (create/delete, cache-hit expectation), auxiliary commands (help/completion/register/mcp-help), and cleanup. GUI/update remain opt-in.
- Shared workflows are defined once in YAML (`internal/validator/workflows/default.yaml`) and executed via both CLI and MCP tools to keep a single source of truth.
- Auto-picks the smallest image/template when none are provided, and cleans up VMs/templates/cache entries after runs.
- Outputs: NDJSON log at `logs/cli-validate-<ts>.ndjson` plus human summary `.summary.txt`. Each step records command, args, exit code, duration, stdout/stderr, assertions, and result (PASS/FAIL/SKIP).

Running
- From repo root: `go run ./cmd/nido-validator` (defaults to `nido` in PATH, auto-builds if missing).
- Optional flags/env:
  - `--nido-bin` / `NIDO_BIN`: path to `nido` binary.
  - `--template` / `NIDO_TPL`: base template for spawn.
  - `--image` / `NIDO_IMAGE`: base image for `--image` spawn and image-pull workflow.
  - `--user-data` / `NIDO_USER_DATA`: cloud-init file; otherwise auto-generated marker file.
  - Port bases: `--ssh-port-base`, `--gui-port-base`, `--fw-port-base`.
  - Timeouts: `--boot-timeout`, `--download-timeout`, `--gui-timeout`, `--port-wait-timeout`.
  - Skip flags (default true for safety): `--skip-gui` / `SKIP_GUI`, `--skip-update` / `SKIP_UPDATE`.
  - Behavior: `--fail-fast` (default true), `--keep-artifacts` (default false), `--workflow` / `NIDO_WORKFLOW` to point to a custom YAML workflow, `--pool-image` / `POOL_IMAGE` to target an image pool pull.

Workflows (single source of truth)
- Default YAML `internal/validator/workflows/default.yaml` contains:
  - Template flow: spawn → template create → spawn from template (cache-hit expectation) → delete spawned VMs → delete template.
  - Image pool flow: pull a configured image, verify cache entry, spawn from image, delete VM, prune cache entry.
- CLI executor runs these steps via CLI; MCP executor starts `nido mcp` and replays with MCP tools (`vm_create`, `vm_template_create/delete`, `vm_delete`, `vm_info`, `vm_template_list`).
- Cache-hit expectation uses backing state and tolerant timing; logs allow further inspection if needed.

What it validates (high level)
- Version/doctor JSON schema and success.
- Images/cache/templates list endpoints (JSON parsable, presence of fields).
- Image pool: pull configured image, cache populated, spawn from image uses cache/backing.
- VM lifecycle: spawn with port mapping, info/list consistency, SSH command succeeds (`ssh -- echo ok` with retries), start/stop/delete/prune exit codes.
- Port forwarding connectivity: optional host dial of forwarded port when dummy server is started via SSH.
- Cloud-init marker: optional check that user-data marker file exists in guest.
- Template workflow: template create/delete succeeds; second spawn from template runs; MCP mirrors these actions.
- Auxiliary: help/completion/register/mcp-help; GUI/update are SKIP by default (enable with flags).
- Cleanup: tracked VMs, templates, temp files removed unless `--keep-artifacts`.

Notes and gaps
- GUI/update remain opt-in; provide `NIDO_UPDATE_URL`/`NIDO_RELEASE_API` stubs for safe runs.
- Forwarding dial and cloud-init checks depend on guest capabilities; enable only if image supports SSH and Python (for dummy server).
