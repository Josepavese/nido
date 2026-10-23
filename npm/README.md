# nido-mcp

npm launcher for [Nido](https://github.com/Josepavese/nido) — fast, local
QEMU/KVM VM sandboxes for AI agents, with a native MCP server. Ideal for
running the MCP server from clients that only know how to launch `npx`:

```bash
npx -y nido-mcp version
npx -y nido-mcp mcp        # stdio MCP server
```

The launcher has **zero npm dependencies**. On first run it downloads the
matching binary from [GitHub Releases](https://github.com/Josepavese/nido/releases),
verifies the SHA-256 against the published `SHA256SUMS`, extracts it into a
cache directory and executes it. Downloads are lazy (no `postinstall`), so
`npm --ignore-scripts`, pnpm 10+ build policies and locked-down registries
keep working.

## Requirements

- Node.js >= 18
- Linux, macOS or Windows (amd64 or arm64)
- QEMU/KVM on the host for actually running VMs (see the
  [main README](https://github.com/Josepavese/nido#readme))

## MCP client configuration

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

## Environment variables

| Variable | Purpose |
| --- | --- |
| `NIDO_NPM_VERSION` | Pin a Nido release version (default: the package version). |
| `NIDO_NPM_CACHE` | Cache directory for downloaded binaries. |
| `NIDO_NPM_VERBOSE=1` | Print download/extraction details to stderr. |
| `NIDO_BINARY` | Use an existing Nido binary instead of downloading. |

## Security

- The binary is verified against the release's `SHA256SUMS` before execution.
- The npm package is published from CI with provenance (trusted publishing),
  so the wrapper can be traced back to this repository and commit.
- Checksums cover the binary download; npm provenance covers the wrapper.

MIT licensed.
