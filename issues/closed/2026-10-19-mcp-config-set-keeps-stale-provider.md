# [P2] MCP config_set leaves the running provider on the previous configuration

Date: 2026-10-19
Status: fixed in v4.5.29; originally reproduced on v4.5.28 in one MCP process, with CLI comparison.

## Reproduction and impact

Send these `nido_system` tool calls sequentially through the same MCP process:

```json
{"action":"config_get"}
{"action":"config_set","key":"SSH_USER","value":"new-audit-user"}
{"action":"config_get"}
```

The first reports the fixture default `audit`. The update succeeds and its
response reports `new-audit-user`. The third reports `audit` again.
The file contains the new value, and a fresh `nido config --json` reads it
correctly. Thus the write succeeds, but the active MCP provider retains the old
configuration. This persists beyond the update response until process restart.

## Cause and required outcome

`internal/mcp/mcp.go:766` writes and reloads a local `cfg` for the response, without
updating the provider's configuration. `config_get` (`:764`) reads
`Provider.GetConfig()`. The process retains two different effective views.

Give configuration refresh one canonical owner and apply it to the active
provider after validated changes. Avoid a concrete-provider cast in MCP or a
second MCP-only configuration store. Define what applies immediately, and test
set/get plus an affected provider operation in the same session. Preserve CLI
fresh-process readback and existing data paths.

Evidence: `mcp-config-stale.json`; safe reproducer case `mcp-config-stale`.
See [audit scope and evidence](../../docs/audits/2026-10-19-cli-mcp.md).

## Resolution, v4.5.29

Fixed in the shared provider/configuration boundary. The tracked regression suite
`tests/cli_mcp_safety.py` exercises both CLI and MCP against isolated synthetic
fixtures; the relevant defect now passes, with preservation and positive controls.
See the audit report for the complete validation scope. No existing VM was used
for qualification and no data conversion or VM-state migration is required.
