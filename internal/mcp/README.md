# MCP implementation

The user-facing setup, tools, resources and operational semantics are documented
in the [MCP guide](../../docs/mcp.md).

`mcp.go` owns the tool/resource catalogs, action dispatch and machine-readable
guide. It consumes `provider.VMProvider`; native device and process behavior stays
behind the HAL/PAL contracts. Keep stdout reserved for the stdio protocol.

When changing this surface, update the owner, the affected public guide and
CLI/MCP parity tests together. `mcp_test.go` checks the exposed catalogs and action
mapping; `tests/catalog_surfaces.py` and `tests/cli_mcp_safety.py` provide isolated
blackbox coverage. See [testing](../../TESTING.md).
