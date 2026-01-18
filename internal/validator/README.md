Nido Validation Engine
======================

What this does
- End-to-end validation of both Nido CLI and MCP: version/doctor, images/cache/templates, VM lifecycle (spawn/info/list/ssh/start/stop/delete/prune), template workflows, image-pool flows, MCP tools parity, auxiliary commands, and cleanup.
- Single source of truth for workflows: YAML at `internal/validator/workflows/default.yaml` is executed twice (CLI and MCP) to keep behavior aligned.
- Auto-picks the smallest image and template when none are provided, so you can run with zero flags; artifacts are cleaned up (VMs, templates, cache entry).
- Outputs machine-readable NDJSON plus a human summary and a live, colored progress HUD on stdout.

How to run
- Fast path (auto build `nido` if missing, auto-pick smallest image/template):
  ```bash
  go run ./cmd/nido-validator --skip-gui --skip-update
  ```
- Recommended (prebuilt binaries to avoid compile overhead):
  ```bash
  go build -o bin/nido ./cmd/nido
  go build -o bin/nido-validator ./cmd/nido-validator
  NIDO_BIN=bin/nido SKIP_GUI=true SKIP_UPDATE=true bin/nido-validator
  ```
- Common flags/env:
  - `--nido-bin` / `NIDO_BIN`: path to `nido` (auto-builds if not found).
  - `--image` / `NIDO_IMAGE`: force base image; otherwise the smallest catalog image is chosen.
  - `--template` / `NIDO_TPL`: force base template; otherwise the smallest available template is chosen.
  - `--pool-image` / `POOL_IMAGE`: image-pool target; defaults to the base/auto image.
  - `--user-data` / `NIDO_USER_DATA`: cloud-init file (else a marker file is generated).
  - Timeouts/ports: `--boot-timeout`, `--download-timeout`, `--gui-timeout`, `--port-wait-timeout`, `--ssh-port-base`, `--gui-port-base`, `--fw-port-base`.
  - Behavior: `--skip-gui`, `--skip-update`, `--fail-fast`, `--keep-artifacts`, `--workflow` / `NIDO_WORKFLOW` for custom YAML, `--check-forwarding`, `--check-cloud-init`.

What gets tested (scenarios)
- Pre-clean: remove stale VMs/templates with test prefixes.
- Pre-flight: `version --json` and `doctor --json` schema/values.
- Images/cache/templates: list + base/pool image pull, cache info/list, template list with auto selection.
- VM lifecycle: spawn (with optional ports/user-data), info/list consistency, SSH echo, start/stop/delete, prune, optional forwarding + cloud-init checks.
- Workflows (from YAML): template flow (spawn → template create → spawn from template → delete VMs → delete template); image-pool flow (pull → spawn from image → delete VM → cache rm). Executed via CLI and again via MCP tools for parity.
- MCP protocol: initialize, tools/list expected set, positive and negative tool calls.
- Auxiliary: help, completion, register, mcp-help; GUI/update are opt-in skips by default.
- Cleanup: tracked VMs/templates and temp files removed unless `--keep-artifacts`.

Outputs
- NDJSON log: `logs/cli-validate-<timestamp>.ndjson` (every step with command, args, exit code, duration, stdout/stderr, assertions, result).
- Summary text: `logs/cli-validate-<timestamp>.summary.txt` with counts and duration.
- Live HUD on stdout: colored progress per step, scenario names, status badges, durations; honors `NO_COLOR`/`TERM=dumb`.

Behavior and defaults
- Auto image/template selection chooses the smallest advertised size when none are provided, and records them for both CLI and MCP workflows.
- Image pulls and cache removals are mirrored between CLI and MCP; templates created during workflows are deleted at the end.
- MCP client tolerates stdout noise by line-buffered JSON parsing; MCP server uses quiet downloads to keep JSON clean.
- Fail-fast stops early on failure (except cleanup), but defaults are tuned for full coverage.

Extending/adding tests
- Add or edit workflows in `internal/validator/workflows/default.yaml` (or point `--workflow` to another YAML) to define shared action sequences.
- New scenarios can be added under `internal/validator/scenario` and wired into `cmd/nido-validator/main.go`.
- Keep CLI and MCP coverage in sync by implementing actions in both `executeWorkflowCLI` and `executeWorkflowMCP`.

Performance tips
- Prebuild `nido` and `nido-validator` to avoid repeated compilation.
- Ensure your chosen image is already cached to skip download time.
- Use a small image (default auto-pick favors the smallest) and keep `--skip-gui/--skip-update` enabled for faster runs.

Logs to inspect
- Latest run log: `logs/cli-validate-*.ndjson`
- Failures are echoed in the final retro summary and can be filtered by assertion names in the NDJSON file.
