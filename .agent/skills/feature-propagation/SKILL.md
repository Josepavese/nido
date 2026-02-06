---
name: feature-propagation
description: Definitive protocol for adding ANY new feature, flag, or command to Nido. Enforces deep integration across Engine, CLI, Config, Info, MCP, TUI, and Docs, with recursive checks for configuration exposure.
---

# Feature Propagation Protocol

This skill enforces a **mandatory, recursive sequence** of steps to ensure every new feature is deeply integrated into the Nido ecosystem.

**The Core Principle**: A feature is not "done" until it is discoverable, configurable, and observable by both Humans (CLI, TUI, Docs) and Machines (MCP, Config, Info).

## The Recursive Check (Start Here)

Before implementing the feature, ask: **"Does this feature significantly alter behavior or require persistence?"**

*   **IF YES**: You must likely expose it via `nido config`.
    *   *Action*: Recursively apply this skill to add the configuration field first.
*   **IF YES**: You must expose the current state via `nido info`.
    *   *Action*: Recursively apply this skill to add the info field.

---

## 1. Engine Layer: The Source of Truth

*   **Mandate**: Update core structures (e.g., `provider.VMOptions`, `config.Config`) to support the new parameter.
*   **Mandate**: Implement the logic in the underlying engine (e.g., `qemu.go`, `manager.go`).
*   **Mandate**: Define sensible defaults and handle invalid input at the lowest level.
*   **Mandate**: If this is a configurable option, ensure `UpdateConfig` supports it.

## 2. Introspection Layer: Visibility & Configurability

*   **Info Mandate**: If the feature adds state/options, expose them in `nido info`.
    *   *Why?* So MCP agents can see the capability exists.
*   **Config Mandate**: If the feature is persistent, expose it in `nido config`.
    *   *Why?* So users and agents can permanently tune the environment.
*   **Autocomplete Mandate**: Update autocomplete logic if the feature introduces new dynamic values (e.g., new enum types).

## 3. CLI Layer: The Interface

*   **Mandate**: Parse the new flag or command in the CLI entry point (e.g., `cmd/`).
*   **Mandate**: Update `usage` or `help` text to document the capability.
*   **Mandate**: Bridge the CLI input to the Engine layer structures.

## 4. Shell Completion: Discoverability

*   **Mandate**: Add the flag/command to the **Bash** completion script.
*   **Mandate**: Add the flag/command to the **Zsh** completion script.
*   **Test**: Run `source <(nido completion bash)` and verify the new flag appears.

## 5. MCP Layer: Robot Interface

*   **Mandate**: Add the field to the `ToolsCatalog` JSON schema in `internal/mcp/mcp.go`.
*   **Mandate**: Map the MCP JSON arguments to the Engine structures.
*   **Validation**:
    *   Run `nido mcp-help` and verify the new field/description is present.
    *   If you added to `nido info` or `nido config`, verify `mcp_server` tools return the new data.
*   **Manifest**: If adding a **new tool**, update `server.json`. (Parameter additions do not require this).

## 6. TUI Layer: Visual Control

*   **Mandate**: Add the new parameter/control to the relevant TUI page or modal.
*   **Mandate**: Ensure TUI state stays in sync with the underlying Engine state.

## 7. Documentation: The Contract

*   **Mandate**: Add the new flag/command to the relevant tables in `README.md`.
*   **Mandate**: Include at least one concrete example showing how to use the feature.

## 8. Validation: Proof of Work (MANDATORY)

*   **STRICT MANDATE**: If you added a new **CLI Parameter** or **Command**, you **MUST** create a new test case in the validator (e.g., `scenario/`).
    *   **NO EXCEPTIONS**: It is forbidden to "trust it works".
*   **Coverage**: The test must cover:
    1.  **Valid Usage**: Confirm the feature works as intended (e.g., verifying `/proc/cmdline` inside the VM via SSH).
    2.  **Invalid Usage**: Confirm the system handles bad inputs gracefully (if applicable).
*   **Execution**: Run the validator locally to confirm the new test passes.
