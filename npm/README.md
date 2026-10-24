# Nido - Local VM manager for AI agents

![Nido: Real VMs. Agent-ready.](https://raw.githubusercontent.com/Josepavese/nido/main/resources/nido-hero.webp)

**Give your AI agent a real virtual machine it can provision, use and delete.**

`nido-mcp` is the npm launcher for [Nido](https://github.com/Josepavese/nido),
a local QEMU VM manager with a built-in **stdio MCP server**, a CLI and an
interactive terminal dashboard. Each VM has its own guest kernel. Cached images,
copy-on-write disks and reusable templates make repeated environments practical.

[Quickstart](https://github.com/Josepavese/nido#press-start) · [MCP guide](https://github.com/Josepavese/nido/blob/main/docs/mcp.md) · [Documentation](https://github.com/Josepavese/nido/blob/main/docs/README.md)

## Connect an MCP client

Add this server to your client's MCP configuration, then reconnect the client:

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

Nido exposes five tools: `nido_vm`, `nido_image`, `nido_template`,
`nido_blueprint` and `nido_system`. Prefer read-only resources such as
`nido://fleet/vms` and `nido://catalog/images` for inspection. Tools use an
`action` field for lifecycle, provisioning and configuration operations.

An example task for your agent:

> Create a disposable Ubuntu LTS VM named `test-lab`, provision my dependencies,
> run the tests and save their output. Delete only that VM when finished;
> preserve other VMs and shared images.

The server can change VM state, configuration, cache and installation files.
Use a trusted client and its approval controls. MCP `ssh` returns connection
arguments; the client needs an execution tool to run guest commands.
See the [MCP guide](https://github.com/Josepavese/nido/blob/main/docs/mcp.md).

## Use Nido in a terminal

Run without a global install:

```bash
npx -y nido-mcp version
npx -y nido-mcp doctor
```

Or install the launcher once:

```bash
npm install -g nido-mcp
nido images update
nido images info ubuntu:lts
nido spawn test-lab --image ubuntu:lts --memory 2048 --cpus 2
nido info test-lab --json

# After guest boot and SSH are ready:
nido ssh test-lab
```

The first image download and guest initialization take time. VM disks persist
until you delete them. Export useful results before removing your disposable VM.
For provisioning, forwarded ports, templates, 9p sharing and safe cleanup, use
the [workflow guide](https://github.com/Josepavese/nido/blob/main/docs/usage.md).

## What gets installed?

The launcher has **zero npm dependencies** and no `postinstall` hook. On first
use it downloads the Nido binary matching the package version from GitHub
Releases, verifies the archive against `SHA256SUMS`, and caches the executable.
Later runs reuse that cache. Installation policies such as `--ignore-scripts`
do not prevent this first-use download.

QEMU is a separate dependency; npm does not install it. The package also does
not bundle a guest OS image. Use `nido doctor` to inspect prerequisites and the
effective host acceleration. Hardware acceleration depends on host support and
permissions; software emulation is available where applicable.

| Requirement | Support |
| --- | --- |
| Node.js | 18 or newer, as declared in the package engines |
| Linux | x64 and arm64 release binaries |
| macOS | Intel and Apple Silicon release binaries |
| Windows | x64 release binary; ARM64 has no native Windows release asset |
| Runtime tools | QEMU to run VMs, SSH for guest connections; tar on Unix / PowerShell extraction on Windows |

[Host setup and other installers](https://github.com/Josepavese/nido/blob/main/installers/README.md)

## Configuration and repeatable installs

| Variable | Purpose |
| --- | --- |
| `NIDO_NPM_VERSION` | Select a Nido release instead of the package version |
| `NIDO_NPM_CACHE` | Directory for downloaded executables |
| `NIDO_NPM_VERBOSE=1` | Show download and extraction details on stderr |
| `NIDO_BINARY` | Use an existing executable instead of downloading |

The launcher defaults to the package's own version. Pin the npm package version
in an automation environment when you need a fixed launcher/release pair.
Checksum verification covers the release archive; npm provenance identifies the
published wrapper. These checks do not replace host or guest security policy.

[Image catalog](https://github.com/Josepavese/nido/blob/main/docs/images.md) · [Windows](https://github.com/Josepavese/nido/blob/main/docs/windows.md) · [Source](https://github.com/Josepavese/nido) · [Issues](https://github.com/Josepavese/nido/issues)

MIT licensed. **Hatch your agents. Let them fly.**
