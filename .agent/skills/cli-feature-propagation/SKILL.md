---
name: cli-feature-propagation
description: Multi-layered protocol for propagating new features across Engine, CLI, MCP (Model Context Protocol), TUI, and Documentation. Use this skill when adding a new flag, command, or capability to the system to ensure it is robustly integrated, discoverable, and AI-ready.
---
# CLI Feature Propagation Protocol

This skill enforces a mandatory sequence of steps to ensure a new feature is fully integrated into the ecosystem. Use this every time you modify the core capabilities of the software.

## 1. Engine layer: modify the source of truth

- **Mandate**: Update core structures (e.g., `provider.VMOptions`) to support the new parameter.
- **Mandate**: Implement the logic in the underlying engine (e.g., `qemu.go`).
- **Mandate**: Define sensible defaults and handle invalid input at the lowest level.

## 2. CLI layer: expose the feature

- **Mandate**: Parse the new flag or command in the CLI entry point (e.g., `main.go`).
- **Mandate**: Update the `usage` or `help` functions to document the new capability.
- **Mandate**: Bridge the CLI input to the Engine layer structures.

## 3. Shell Completion: ensure discoverability

- **Mandate**: Add the flag/command to the **Bash** completion script (e.g., `getBashCompletion`).
- **Mandate**: Add the flag/command to the **Zsh** completion script (e.g., `getZshCompletion`) with a clear description.

## 4. MCP layer: publish to AI Agents

- **Mandate**: Add the field to the `ToolsCatalog` JSON schema in `internal/mcp/mcp.go`.
- **Mandate**: Map the MCP JSON arguments to the Engine structures in the tool handlers.
- **Mandate**: **CRITICAL**: Verify that `nido mcp-help` correctly publishes the new field and its description.

## 5. TUI layer: provide visual control

- **Mandate**: Add the new parameter/control to the relevant TUI page or modal.
- **Mandate**: Ensure the TUI state is kept in sync with the underlying VM data.

## 6. Documentation: update README

- **Mandate**: Add the new flag/command to the relevant tables in `README.md`.
- **Mandate**: Include at least one concrete example showing how to use the feature.

## 7. Validation: end-to-end testing

- **Mandate**: Add a new test step to the validator (e.g., `scenario/vm_lifecycle.go`).
- **Mandate**: Verify functionality by asserting state changes (e.g., checking `/proc/cmdline` via SSH).
