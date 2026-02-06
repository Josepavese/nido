---
name: tui-kit-enforcement
description: Enforces strict adherence to internal/tui/kit as the Single Source of Truth for TUI components. Use when modifying, extending, or creating TUI features.
---

# TUI Kit Enforcement

This skill enforces the rule that `internal/tui/kit` is the **Single Source of Truth (SSOT)** for all TUI components in Nido.

## Core Directives

1.  **SSOT Supremacy**: The directory `internal/tui/kit` is the **ONLY** source for components such as:
    *   Forms (`form.go`, `form_modal.go`)
    *   Inputs
    *   Buttons
    *   Navigation
    *   Headers/Status Bars (`status_bar.go`)
    *   Cards (`card.go`)
    *   Modals (`modal.go`, `list_modal.go`)
    *   Lists (`list.go`, `sidebar_list.go`)
    *   Layouts (`master_detail.go`, `split_view.go`, `boxed_sidebar.go`)
    *   *See `internal/tui/kit/widget` for the exhaustive list.*

2.  **No Reinvention**: You must **NOT** reinvent or recreate any component that already exists in the kit.
    *   **Always** first check `internal/tui/kit` for an existing component that fits the need.
    *   **Copy** existing implementation patterns found elsewhere in the codebase (search for usages of `kit.*`).

3.  **Behavior Encapsulation**: The behavior of an element must be encoded **WITHIN** the kit.
    *   **Reject** custom, one-off behaviors implemented in specific views that diverge from the kit's standard.
    *   All components must behave consistently across the application (SSOT).

## Protocol for New Elements

If and **ONLY IF** a required UI element does NOT exist in the kit:

1.  **MANDATORY User Interaction**: You must **STOP** and consult with the user.
    *   Propose the new component.
    *   Confirm the design and interaction model.
    *   **Do not proceed** without explicit confirmation.

2.  **Strict Consistency**: The new element must rigorously follow the graphic line and code style of existing `kit` elements.
    *   Use the existing `theme` definitions (`internal/tui/kit/theme`).
    *   Follow the API patterns of existing widgets (e.g., `Update`, `View`, `Model`).

3.  **Integration into Kit**: The new component must be developed **inside** `internal/tui/kit`, making it available for reuse immediately. It should not be defined locally in a specific feature package.

## References

*   **Single Source of Truth**: Follow the principles in `../../skills/single-source-of-truth/SKILL.md`.
*   **Kit Directory**: `internal/tui/kit`
