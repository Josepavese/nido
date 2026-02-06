# Nido CLI JSON Output Specification

## Purpose

This document defines the **stable JSON output** format for Nido CLI commands used by automation and the future GUI.
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
- `ERR_NOT_FOUND`
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
- `delete`
- `prune`
- `template list|create|delete`
- `image list|pull|info|remove|update`
- `cache ls|info|rm|prune`
- `version`
- `doctor`
- `config`
- `register`

## Payload Shapes (Summary)

### `ls`

`data.vms[]`: name, state, pid, ssh_port, vnc_port, ssh_user

### `info`

`data.vm`: name, state, ip, ssh_user, ssh_port, vnc_port, raw_qemu_args

### `spawn|start|stop|delete|prune`

`data.action` or `data.removed_count`

### `template list`

`data.templates[]`: name, size_bytes

### `image list`

`data.images[]`: name, version, registry, size_bytes, aliases, downloaded

### `image pull|update`

`data.action`: name, version, result

### `cache ls`

`data.cache[]`: name, version, size_bytes, modified_at

### `cache info`

`data.stats`: total_images, total_size, oldest, newest

### `doctor`

`data.reports[]`: raw diagnostic lines  
`data.summary`: total, passed, failed

### `config`

`data.config_path`, `data.backup_dir`, `data.default_tpl`, `data.ssh_user`, `data.linked_clones`

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
