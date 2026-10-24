# Nido documentation

Nido manages local QEMU VMs through the CLI, terminal dashboard and stdio MCP.
Start with the [README quickstart](../README.md#press-start), then choose a guide.

## Use Nido

| Guide | What it covers |
| --- | --- |
| [Installation](../installers/README.md) | Binary installers, package managers, host prerequisites and upgrades |
| [Everyday workflows](usage.md) | Provisioning, SSH, ports, templates, raw QEMU arguments, sharing and cleanup |
| [MCP](mcp.md) | Client configuration, tools, resources and CLI/MCP differences |
| [Images and blueprints](images.md) | Catalog discovery, aliases, integrity, cache and image recipes |
| [Windows](windows.md) | Host qualification, WHPX setup and Windows evaluation guests |
| [Agent skill](agent-skills.md) | Installable VM-testing skill and repeatable test environments |
| [CLI JSON](CLI_JSON_SPEC.md) | Supported commands, response envelope and payloads |
| [Themes](customization.md) | Custom terminal palettes |

## Develop and maintain

- [Contributing](../CONTRIBUTING.md) and [testing](../TESTING.md)
- [Runtime architecture](ARCHITECTURE.md), [CLI command manifest](CLI_ARCHITECTURE.md) and [I/O boundaries](IO_BOUNDARIES.md)
- [Maintainer tools](MAINTAINERS.md) and [distribution channels](DISTRIBUTION.md)
- [Roadmap](ROADMAP.md) and [changelog](../CHANGELOG.md)
- [CLI/MCP audit, 2026-10-19](audits/2026-10-19-cli-mcp.md) - historical findings and evidence

## Keep the guides current

The command manifest (`internal/cli/commands.yaml`), MCP catalog
(`internal/mcp/mcp.go`) and registry source policies (`registry/sources.yaml`)
own their respective contracts. `nido help`, command-specific `--help` and
`nido mcp-help` describe the installed binary. The README is the introduction;
these guides hold the details. The [GitHub wiki](https://github.com/Josepavese/nido/wiki)
is a navigation entry point to these versioned pages, not a second operations manual.

When changing a command, update its owner and affected guide together. Prefer
catalog aliases for introductory examples and pin numbered tags for a specified
OS release. Keep measured qualification separate from claims about all hosts or
guest images. Describe supported behavior before using arcade metaphors, and use
the ASCII hyphen `-` for prose separators.
