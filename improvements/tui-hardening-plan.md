# Nido TUI Hardening & De-hardcode Plan

Goal: eliminate hardcoded layout/style/logic, make the TUI fully declarative and configurable, and stabilize visuals across all views.

## 1) Audit hardcoded values
- Scan TUI for fixed numbers and strings: padding/gaps/widths/heights, colors, labels, shortcuts, footer links, image/cache paths.
- Key files: `internal/tui/gui/*`, `layout/*`, `viewlet/*`, `components/*`, `theme/*`.
- Produce an audit table (value → usage → proposed token/config).

## 2) Centralize design tokens
- Spacing: a single scale in `theme/spacing.go` (XS/SM/MD/LG). Replace inline `strings.Repeat`/magic gaps with these tokens.
- Widths/heights: `theme/width.go` (sidebar regular/wide, label width, tab min width, content padding). No `-4`, no `18/28` inline.
- Colors: only via palette; remove inline color literals.
- Footer/header/subheader heights and gaps: define once in `layout/constants.go`.

## 3) Unified shell composition
- Use `RenderShell` as the sole source for header/subheader/footer; it should also return actual heights/gap used.
- `model.View`: (a) `dim := layout.Calculate(...)`; (b) `header, sub, footer, shellH := RenderShell(...)`; (c) `bodyHeight := dim.Height - shellH`; (d) mount viewlet.
- Compose with `layout.VStack` instead of manual `"\n\n"` concatenation.

## 4) Declarative, responsive layout
- Breakpoints in `layout/breakpoints.go` with configurable thresholds; sidebar widths/content widths per breakpoint come from tokens/config.
- H/VStack helpers use spacing tokens; no ad-hoc gaps.
- Containers (cards/panels/status bar) use shared padding/width constants; apply truncation/MaxWidth instead of relying on wrap.

## 5) Component configurability
- Table, Tabs, StatusBar, Modal, Form, Badge accept config structs (padding, borders, show-exit). Defaults pulled from theme tokens.
- No embedded numbers inside components; defaults live in one place.

## 6) Keymap and strings
- Centralize keybindings in `gui/keymap.go` (with overrides allowed via config/env).
- Centralize user-visible strings (tab labels, hints, footer link) in `gui/strings.go`; allow overrides where sensible.

## 7) External configuration
- Add `TUI` section to `config.Config` with: theme mode (auto/light/dark), breakpoint overrides, sidebar widths, gap scale, keymap overrides, footer link, mouse enable, show exit button.
- Env overrides: `NIDO_THEME`, `NIDO_TUI_BREAKPOINT_*`, `NIDO_TUI_SIDEBAR_*`, `NIDO_TUI_GAP_SCALE`, key overrides.
- Load once in `initialModel` and pass through to shell/layout/components/viewlets.

## 8) Viewlet contract
- Each viewlet gets `Resize(bodyW, bodyH)` and a `SetConfig(TUIConfig)`; no internal numbers—use tokens/helpers.
- Shell elements (header/subheader/footer) live only in shell; viewlets focus on body content.

## 9) Testing/stability
- Align `layout.Calculate` tests with the single overhead definition; add snapshots for `RenderShell` at widths 80/120/160 to catch spacing regressions.
- Add tests for config/env overrides (theme light/dark, breakpoint overrides).

## 10) Execution order
1) Decide and lock the overhead (header/subheader/footer+gaps) → update `layout.Calculate` and tests.
2) Refactor `View()` to use `RenderShell`/`layout.VStack`; remove manual gaps and inline heights.
3) Introduce `theme/width.go` and replace all magic widths/heights/gaps with tokens.
4) Add `TUIConfig` + env overrides; propagate to shell/layout/components/viewlets.
5) Centralize keymap/strings.
6) Cleanup pass removing residual `-4`, `"\n\n"`, inline spacers.
