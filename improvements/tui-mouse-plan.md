# TUI Mouse & Focus Handling Plan

Context: recent refactors centralized shell/layout/theme, but mouse click/focus for tabs/sidebars/forms regressed. This plan outlines a structured, low-drift approach to restore and harden mouse/focus handling.

## Goals
- Reliable click-to-select for top tabs/exit, sidebars (Fleet/Hatchery/Config), tables/lists, and form fields.
- Keep hit-testing in sync with rendering to avoid drift.
- Minimize duplication and keep routing simple.

## Key Principles
- Derive hitboxes from the same measurements used to render (real widths/heights with padding/borders), not approximations.
- Delegate interactions to viewlets/components; the root only translates coordinates.
- Recompute hit maps on resize/theme/config changes.
- Prefer existing Bubbles mouse support where available (list/table).

## Proposed Architecture

1) **Shell hit-map**
   - After calling `RenderShell`, compute tab widths and exit-zone width using the same inputs (labels, `theme.Width.TabMin`, `theme.Width.ExitZone`, actual rendered header height).
   - Store a `ShellHitMap` (tab ranges + exit zone). Use it in `handleHeaderMouse` to switch tabs/quit. Recompute on resize or tab label change.

2) **Global dispatcher**
   - A single mouse dispatcher in `model`:
     1) Header: consult `ShellHitMap` first.
     2) Sidebar region: if visible, delegate click to the active sidebar component/viewlet.
     3) Body: translate global `(x,y)` to body-local coords and delegate to the active viewlet’s `HandleMouse`.
   - Short-circuit on handled events; no duplicated logic.

3) **Viewlet mouse interface (optional but recommended)**
   - Extend `Viewlet` with `HandleMouse(x, y tea.MouseMsg) (Viewlet, tea.Cmd, bool)` where `(x,y)` are body-relative.
   - Each viewlet is responsible for internal focus/selection; root only does coordinate translation.
   - For components (list/table), prefer their native `Update` with mouse enabled and correct sizes.

4) **Sizing alignment**
   - Ensure sidebar/list/table/viewlet sizes are set from the same tokens used in rendering (`layout.Calculate`, `theme.Width`, `theme.Inset`, `theme.Gap`).
   - Avoid magic offsets; include borders/padding when computing hitboxes.

5) **Resilience to overrides**
   - Hit-map recomputation on:
     - `tea.WindowSizeMsg`
     - Tab label changes (config/env)
     - Theme/layout overrides (sidebar widths, exit zone, tab min width)
   - Store hit-map in model state; invalidate/rebuild on the above events.

6) **Testing**
   - Add a test that renders a shell with known widths and verifies tab/exit hit ranges are non-empty and within bounds.
   - Optional: property test to ensure tab click -> correct `activeTab` at several widths/breakpoints.
   - For viewlets, rely on Bubbles’ tested mouse support where possible; add a small test for a custom `HandleMouse` if implemented.

## Risks / Mitigations
- Drift between render and hit-test: mitigate by deriving from actual render inputs and shared tokens; avoid hand-coded offsets.
- Mouse double-handling: ensure dispatcher stops propagation once handled; enable Bubbles mouse support only where intended.
- Maintenance overhead: keep hit-map small and recomputed in one place; document the contract in `gui/README.md`.

## Minimal Implementation Steps
1) Add `ShellHitMap` struct and compute it alongside `RenderShell` outputs; use it in header mouse handling.
2) Add a root-level dispatcher ordering: header → sidebar → viewlet body (with coordinate translation).
3) Optionally extend `Viewlet` with `HandleMouse`; wire Fleet/Hatchery/Config to delegate to their list/form widget.
4) Add a small shell hit-range test; document mouse handling in `improvements/TUI_CONFIG.md` or `gui/README.md`.
