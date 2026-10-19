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

For `config_update`, `raw_qemu_args` replaces the saved argument list, preserving
order and individual strings. Omit it to retain the current list, or send `[]`
to clear it. `null` and non-array values are rejected. Changes are saved for the
next start, including when the VM is stopped; they do not hot-plug devices.

```json
{"action":"config_update","name":"agent-01","raw_qemu_args":[]}
```

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

### Mutation and inspection guarantees

VM/template/image names are identifiers, never filesystem paths. VM info returns
errors for nonexistent VMs and corrupt state; disk-only recovery is explicitly
marked by `state_missing`. CLI JSON exposes the same disk/backing health.
Image removal uses the shared provider usage guard, including complete relative
and transitive backing chains. Inspection failures prevent destructive cleanup.
`config_set` refreshes the active provider before returning, so following calls
in the same MCP session use the saved configuration. Automatic forwarding ports
configured on a stopped VM are resolved and persisted before its next launch.
VM stop verifies process identity; an unrelated persisted PID is refused.
