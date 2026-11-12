# Nido CLI JSON Output Specification

## Purpose

This document defines the **stable JSON output** format for Nido CLI commands used by automation and agent clients.
Human output remains unchanged unless `--json` is passed.

## Activation

Add the `--json` flag to supported commands:

```bash
nido ls --json
nido info my-vm --json
```

## Envelope

All JSON responses share a single envelope:

```json
{
  "schema_version": "1.0",
  "command": "ls",
  "status": "ok",
  "timestamp": "2026-01-08T12:00:00Z",
  "data": {}
}
```

Fields:

- `schema_version`: Contract version (SemVer).
- `command`: The CLI command or subcommand.
- `status`: `ok` or `error`.
- `timestamp`: RFC 3339 timestamp (UTC).
- `data`: Result payload for success.
- `error`: Error object for failures.

## Error Object (RFC 7807‑inspired)

When `status=error`, the response includes an error object:

```json
{
  "type": "about:blank",
  "title": "VM not found",
  "detail": "No VM named 'ghost-vm' exists.",
  "code": "ERR_NOT_FOUND",
  "hint": "Check the VM name and try again."
}
```

Fields:

- `type`: Problem type URI (default `about:blank`).
- `title`: Short, human‑readable summary.
- `detail`: Human‑readable explanation.
- `instance`: Optional identifier for this occurrence.
- `code`: Stable error code.
- `hint`: Optional actionable suggestion.
- `details`: Optional debug payload.

### Error Codes (Initial Set)

- `ERR_INVALID_ARGS`
- `ERR_INVALID_ARGUMENT`
- `ERR_NOT_FOUND`
- `ERR_NOT_READY`
- `ERR_WAIT_FAILED`
- `ERR_SHUTDOWN_FAILED`
- `ERR_SNAPSHOT_FAILED`
- `ERR_IO`
- `ERR_DEPENDENCY`
- `ERR_PERMISSION`
- `ERR_INTERNAL`
- `ERR_NOT_IMPLEMENTED`

## Supported Commands

The following commands support `--json`:

- `ls`
- `info`
- `spawn`
- `start`
- `stop`
- `shutdown`
- `delete`
- `prune`
- `snapshot create|list|restore|delete`
- `template list|create|delete`
- `image list|pull|info|remove|update`
- `blueprint list|info|build`
- `cache ls|info|rm|prune`
- `version`
- `doctor`
- `accel list`
- `config`
- `register`

## Payload Shapes (Summary)

### `ls`

`data.vms[]`: name, state, pid, ssh_port, vnc_port, ssh_user, readiness (optional observed guest state)

### `info`

`data.vm`: name, state, ip, ssh_user, ssh_port, vnc_port, raw_qemu_args, readiness (optional observed guest state)

### `wait`

`data.action`: name and result (`ready`).
`data.readiness`: `ready` (boolean), `state`, `phase`, `pid`, `attempts`,
`provisioning` (`absent`, `running`, `done`, `degraded` or `error`), optional
`failure`, optional bounded `detail`, `checked_at` (RFC 3339 UTC) and
`duration_ms`.

A guest that is not ready before the timeout is reported as `status: error` with
code `ERR_NOT_READY`; the readiness observation is in `error.details` and the
command exits non-zero. An invalid timeout uses `ERR_INVALID_ARGUMENT`; an
infrastructure failure uses `ERR_WAIT_FAILED`.

### `spawn|start|stop|delete|prune`

`data.action` or `data.removed_count`

### `shutdown`

`data.action`: name and result (`stopped`).
`data.shutdown`: `name`, `mode` (`guest` or `already_stopped`), optional `pid`,
and `duration_ms`.

`mode=guest` means the guest completed a requested poweroff and the owned QEMU
process exited. `already_stopped` means there was no running guest to ask.
Timeout or guest failure is `status: error` with code `ERR_SHUTDOWN_FAILED`; the
VM is not force-terminated. An invalid timeout uses `ERR_INVALID_ARGUMENT`.

```bash
nido shutdown agent-01 --timeout 120 --json
```

### `snapshot create|list|restore|delete`

All snapshot operations require a stopped VM and return code
`ERR_SNAPSHOT_FAILED` otherwise.

- `snapshot create`: `data.action` with `name`, `snapshot` and `result`
  (`created`); `data.snapshot` is the created `{id,name,vm_size,created_at}`.
- `snapshot list`: `data.action` and `data.snapshots[]`; each entry has `id`,
  `name`, optional `vm_size` and `created_at`. Empty lists are encoded as `[]`.
- `snapshot restore`: `data.action` with `result=restored`.
- `snapshot delete`: `data.action` with `result=deleted`.

`created_at` is `qemu-img`'s local date/time string. Snapshot IDs and dates are
informational; use the snapshot `name` for restore and delete. Snapshots are
disk-only and do not include guest memory or runtime metadata.

### `template list`

`data.templates[]`: template names as strings. Empty lists are encoded as `[]`, not `null`.

### `image list`

`data.images[]`: name, display_name, version, registry, kind (`image` or `blueprint`), size_bytes, aliases, downloaded, output_tag

### `image pull|update`

`data.action`: name, version, result

### `blueprint list`

`data.blueprints[]`: name, display_name, description, version, source, output_image, output_tag, output_path, built

### `blueprint info`

`data.blueprint`: name, display_name, description, version, source, path, output_image, output_tag, output_path, built, cpu, memory, timeout

### `blueprint build`

`data`: result (`built` or `ready`), blueprint, output_image, output_tag, output_path

### `cache ls`

`data.cache[]`: name, version, size

### `cache info`

`data.stats`: total_images, total_size, oldest, newest

### `doctor`

`data.reports[]`: raw diagnostic lines  
`data.summary`: total, passed, failed

### `config`

`data.config_path`: configuration file path.
`data.config`: `backup_dir`, `ssh_user`, `image_dir`, `linked_clones`, `theme`,
`port_range_start`, `port_range_end`, and the `tui` layout object.

### `register`

`data.mcpServers`: MCP configuration block

## Compatibility Rules

- **Non‑JSON output is unchanged** unless `--json` is passed.
- Additive fields are allowed in minor schema versions.
- Breaking changes require a major schema version bump.

## References

- CLI Guidelines: <https://clig.dev/>
- RFC 7807 (Problem Details): <https://www.rfc-editor.org/rfc/rfc7807>
- RFC 3339 (Timestamps): <https://www.rfc-editor.org/rfc/rfc3339>
- JSON Schema: <https://json-schema.org/>

## Event stream

`nido events --json` is a streaming command. It writes NDJSON: each line is a
complete standard response envelope with `command: "events"` and one event in
`data`. Read line by line rather than parsing stdout as one JSON document.
`data` carries version, epoch, sequence, timestamp, source, kind, optional
operation identity/action, VM, resource URIs and a snapshot marker. A resync
barrier requires resource refresh; snapshots use its sequence watermark.
The stream ends on Ctrl+C. See [event semantics](events.md).

VM list/detail and catalog entries include `arch` (`amd64` or `arm64`). VM detail
includes `virtualization_capability` with the HAL accelerator and CPU model;
this is capability information, not a runtime QMP observation. Native image
resolution accepts omitted/`auto` architecture; explicit variants never silently
fall back to another ISA. Automatic state preparation is silent on success and
preserves the normal JSON/MCP protocol. `migrate --json` remains an explicit
adoption/recovery operation and reports the number of records it converts.
