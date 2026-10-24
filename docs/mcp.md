# Connect Nido over MCP

[Documentation](README.md) · [Everyday workflows](usage.md) · [JSON contract](CLI_JSON_SPEC.md)

Nido runs an MCP server over **stdio**, using the same VM provider as the CLI
and TUI. The MCP client starts `nido mcp` as a local child process. There is no
remote HTTP/SSE MCP listener. TUI/MCP sessions share a private local event socket
behind the stdio server; it is not an additional remote MCP transport.

## Choose an installation method

Install the CLI through npm (`nido-mcp`), Homebrew, Scoop or the binary installers,
or use `npx -y nido-mcp mcp` directly from your client. Registry-aware clients
can discover `io.github.Josepavese/nido` in the Official MCP Registry; Nido is
also listed on Glama and ships an MCPB bundle.
See [installation channels and MCP catalogs](../installers/README.md#mcp-catalogs-and-client-installation),
including [WinGet availability](../installers/README.md#winget-windows).

## Configure a client

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

Use an absolute path to the installed binary when the client cannot resolve
`nido` on PATH. `nido register` prints a registration block; it does not install
that block into every client. Use the configuration location documented by your
client, then reconnect it.

For an npm-launching client:

```json
{
  "mcpServers": {
    "nido": {
      "command": "npx",
      "args": ["-y", "nido-mcp", "mcp"]
    }
  }
}
```

The npm package downloads the matching release binary on first use and verifies
its checksum. A local installed executable avoids that first-use download.
The public server identifier is `io.github.Josepavese/nido` in the
[MCP Registry](https://registry.modelcontextprotocol.io/v0/servers?search=io.github.Josepavese/nido).

## Inspect before changing state

Prefer read-only resources where your client supports them:

| Resource | Content |
| --- | --- |
| `nido://fleet/vms` / `nido://vm/{name}` | Fleet and one VM's state |
| `nido://fleet/templates` | Reusable templates |
| `nido://catalog/images` / `nido://image/{tag}` | Image catalog and resolved image metadata |
| `nido://catalog/blueprints` / `nido://blueprint/{name}` | Buildable image recipes |
| `nido://storage/cache` | Cache entries and statistics |
| `nido://system/events` | Stream health, session subscriptions and bounded recent events |
| `nido://system/config` | Effective configuration |
| `nido://system/doctor` | Host diagnostics |
| `nido://system/version` | Version and protocol |
| `nido://system/accelerators` | PCI candidates and safety information |
| `nido://system/mcp-registration` | Registration block |

The tools accept an `action` field. They provide inspection fallbacks as
well as mutations:

| Tool | Actions |
| --- | --- |
| `nido_events` | `status`, `subscribe`, `unsubscribe`, `mute`, `unmute` (session controls) |
| `nido_vm` | `list`, `info`, `create`, `start`, `stop`, `delete`, `ssh`, `prune`, `config_update`, `port_forward`, `port_unforward`, `port_list` |
| `nido_image` | `list`, `info`, `pull`, `remove`, `refresh_catalog`, `cache_list`, `cache_info`, `cache_remove`, `cache_prune` |
| `nido_template` | `list`, `create`, `delete` |
| `nido_blueprint` | `list`, `info`, `build` |
| `nido_system` | `doctor`, `version`, `update_check`, `update`, `config_get`, `config_set`, `accel_list`, `register`, `completion`, `build_image`, `uninstall` |

`nido mcp-help` emits the current machine-readable guide with schemas, examples,
resources and routing rules. `nido mcp --help` is its human-readable entry point.
The `nido_task_router` prompt helps clients choose resources and tool actions.

## Create and configure a VM

These are **tool arguments**, not complete JSON-RPC messages:

```json
{"action":"create","name":"test-lab","image":"ubuntu:lts","memory_mb":2048,"vcpus":2}
```

Send to `nido_vm`, then inspect `nido://vm/test-lab`. Image downloads and guest
boot can take time. CLI `--user-data` reads a file; MCP `user_data` contains the
actual provisioning text. VM creation does not guarantee immediate SSH readiness.

Persistent raw QEMU settings follow replacement semantics:

```json
{"action":"config_update","name":"test-lab","raw_qemu_args":["-fw_cfg","name=opt/example,string=hello world"]}
```

Omit `raw_qemu_args` to keep the saved list; send `[]` to clear it. `null` and
non-arrays are errors. Argument order and string boundaries are preserved.
Updates take effect on the next start, not by hot-plugging a running VM.

## SSH and safety boundaries

MCP `nido_vm` / `ssh` returns connection information, structured `argv` and a
quoted command representation. It does **not** execute arbitrary guest commands
for the client. The client needs an execution tool to use that connection.
CLI `nido ssh <name> ...` executes the local SSH client.

The server can mutate the user's VM state, cache, configuration and installation.
`prune` affects all stopped VMs; `uninstall` requires `force=true`. Client approval
policies remain the client's responsibility. Prefer unique task-owned VM names,
inspect before deleting, and export useful results before cleanup.

Invalid names and corrupt state are errors. VM stop verifies process identity;
image removal checks backing chains through the shared provider. These guards
reduce specific risks; they do not replace a policy for untrusted agents.

## Troubleshooting

- **Executable not found**: use its absolute path and check `nido version` there.
- **Protocol errors**: run the server as a stdio child; do not wrap it with a
  shell profile that writes banners to stdout.
- **Old tool schemas after upgrade**: reconnect the client so it starts the
  current binary and reloads the catalog of tools.
- **Guest not reachable**: inspect VM state and forwarded ports; allow cloud-init
  to finish and check host prerequisites with `nido doctor`.

## Subscribe without notification noise

There are no unsolicited notifications. Subscribe to `nido://vm/test-lab` or a
core resource with standard `resources/subscribe`, or call `nido_events`:

```json
{"action":"subscribe","uri":"nido://vm/test-lab","kinds":["vm.stopped","vm.panic","operation.failed"]}
{"action":"mute","uri":"nido://vm/test-lab"}
{"action":"unmute","uri":"nido://vm/test-lab"}
{"action":"unsubscribe","uri":"nido://vm/test-lab"}
```

Notifications are standard `notifications/resources/updated` with only `uri`.
Repeated changes are coalesced; muted scopes remain quiet. Unmute asks the
client to re-read once, recovering changes it suppressed. Subscriptions end with
the MCP session. The client controls how notifications reach an agent; resource
notifications do not themselves guarantee that a host wakes its language model.

Read `nido://system/events` for health, subscriptions and bounded recent
interested events. On reconnect/resync, re-read resources: the stream is
advisory and does not replay a durable history. See the [event catalog and
lifetime contract](events.md), including process termination versus QMP shutdown.
