# Nido MCP Server

Nido exposes a compact MCP surface designed for agents, not humans.

## Design

The server follows two rules:

- Use a small number of high-power tools for state changes.
- Use resources for read-only inspection whenever possible.

This keeps tool selection simple, cuts prompt token cost, and matches current agent-tool guidance better than a long list of single-action tools.

## Tools

### `nido_vm`

Unified VM tool. Actions:

- `list`
- `info`
- `create`
- `start`
- `stop`
- `delete`
- `ssh`
- `prune`
- `config_update`
- `port_forward`
- `port_unforward`
- `port_list`

`create` accepts the CLI spawn surface exposed to agents: image or template source, user-data content, GUI/cmdline overrides, memory/vCPU sizing, raw QEMU args, accelerators, explicit port mappings, and `web`/`ftp` default forwards. Local images produced by blueprints are resolved from the configured image directory and inherit blueprint SSH/seed metadata.

### `nido_template`

Template management. Actions:

- `list`
- `create`
- `delete`

### `nido_image`

Image catalog and cache management. Actions:

- `list`
- `info`
- `pull`
- `remove`
- `refresh_catalog`
- `cache_list`
- `cache_info`
- `cache_remove`
- `cache_prune`

### `nido_system`

System-wide operations. Actions:

- `doctor`
- `version`
- `update_check`
- `update`
- `config_get`
- `config_set`
- `accel_list`
- `register`
- `completion`
- `build_image`
- `uninstall`

`update`, `config_set`, and `uninstall` mutate the host Nido installation or global config. `uninstall` requires `force=true`.

## Resources

Fixed resources:

- `nido://fleet/vms`
- `nido://fleet/templates`
- `nido://catalog/images`
- `nido://catalog/blueprints`
- `nido://storage/cache`
- `nido://system/config`
- `nido://system/doctor`
- `nido://system/version`
- `nido://system/accelerators`
- `nido://system/mcp-registration`

Parameterized resource templates:

- `nido://vm/{name}`
- `nido://image/{tag}`
- `nido://blueprint/{name}`

## Prompt

The server also exposes one helper prompt:

- `nido_task_router`

It tells MCP-aware clients to prefer resources for inspection and tools for mutations.

## CLI Helpers

- `nido register` prints the MCP registration block
- `nido mcp --help` prints the human-readable MCP guide
- `nido mcp-help` prints the machine-readable MCP guide as JSON

## Example

```json
{
  "mcpServers": {
    "nido": {
      "command": "nido",
      "args": ["mcp"]
    }
  }
}
```
