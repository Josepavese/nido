# Nido TUI Package

This directory contains the refactored Terminal User Interface for Nido.

## Architecture

```text
internal/tui/
├── theme/          # Design system (colors, spacing, typography)
├── layout/         # Declarative layout helpers (grid, breakpoints)
├── components/     # Reusable UI components
├── validators/     # Input validation functions
├── viewlet/        # Viewlet interfaces and implementations
└── gui/            # Main TUI model and entry point
```

## Key Packages

### theme/

Centralized design tokens:

- `palette.go` — Adaptive colors (dark/light + 256c fallback)
- `spacing.go` — Space scale, radius, widths
- `typography.go` — Text style factories
- `theme.go` — Detection and `NIDO_THEME` override

```go
t := theme.Current()
style := lipgloss.NewStyle().Foreground(t.Palette.Accent)
```

### layout/

Declarative layout:

- `HStack(gap, items...)` — Horizontal arrangement
- `VStack(gap, items...)` — Vertical stacking
- `Detect(width)` — Breakpoint detection
- `Calculate(w, h)` — Responsive dimensions

### components/

Reusable widgets:

- `Table` — bubbles/table adapter with Nido theming
- `Badge` — Status indicators
- `StatusBar` — Footer with keymap hints
- `Toast` — Ephemeral notifications
- `Form` — Input fields with validation
- `Modal` — Dialog overlays

### validators/

Input validation:

- `VMName(s)` — 1-32 chars, alphanum with hyphens
- `TemplateName(s)` — Same as VMName
- `FilePath(s)` — Exists and readable
- `Port(n)` — 1-65535

### viewlet/

View implementations (foundation for future refactoring):

- `Viewlet` — Interface for all views
- `Fleet` — VM list with table and detail panel
- `Hatchery` — Spawn/template form
- `Help` — Keyboard shortcuts

> Note: Viewlets are created but not yet fully wired into `model.go`.
> Current TUI uses shell.go helpers (RenderTabs, RenderSubHeader, RenderFooter).

## Environment Variables

| Variable     | Values               | Default |
|--------------|----------------------|---------|
| `NIDO_THEME` | `light`, `dark`, `auto` | `auto`  |

## Running

```bash
nido gui          # Launch TUI
make tui-demo     # Development mode
make test         # Run all tests
```
