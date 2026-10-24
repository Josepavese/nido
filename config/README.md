# Nido configuration

Inspect effective settings before changing storage locations:

```bash
nido config --json
nido config set SSH_USER vmuser
```

The CLI loads `~/.nido/config.env`; if absent, it checks `config/config.env`
relative to the working directory. It does not support a `NIDO_CONFIG` override.
Create the user config through `nido config set` or adapt
[`config.example.env`](config.example.env). Unsupported keys are rejected by the
CLI/MCP setter; the accepted key list is owned by
`internal/config/config.go:SupportedGlobalConfigKeys`.

| Key | Meaning |
| --- | --- |
| `BACKUP_DIR` | Compressed template storage; not an automatic second backup of every VM |
| `IMAGE_DIR` | Downloaded image cache |
| `SSH_USER` | Fallback guest login user; image metadata can select a specific user |
| `LINKED_CLONES` | Use copy-on-write backing disks (default: true) |
| `THEME` | Active terminal palette |
| `PORT_RANGE_START`, `PORT_RANGE_END` | Automatic forwarded service-port range |
| `TUI_*` | Supported layout overrides, listed by config completion |

Changing a path updates configuration, not the location of existing data.
Do not move/delete backing images while dependent disks exist. Preserve useful
VMs, templates and configuration before a deliberate storage migration.
See [everyday workflows](../docs/usage.md) and [themes](../docs/customization.md).
