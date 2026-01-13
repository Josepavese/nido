# TUI Configuration and Overrides

This document describes how to tune the Nido TUI layout, spacing, labels, and keybindings via config file and environment variables.

## Config file keys (`config/nido.cfg`)

```
# Layout
TUI_SIDEBAR_WIDTH=18
TUI_SIDEBAR_WIDE_WIDTH=28
TUI_INSET_CONTENT=4
TUI_TAB_MIN_WIDTH=6
TUI_EXIT_ZONE_WIDTH=4
TUI_GAP_SCALE=1

# Labels / links
TUI_FOOTER_LINK=https://github.com/Josepavese
TUI_TAB_LABELS=1 FLEET,2 HATCHERY,3 LOGS,4 CONFIG,5 HELP
```

## Environment overrides

Each config key can be overridden by an env var (prefix `NIDO_TUI_`), e.g.:

```
export NIDO_TUI_SIDEBAR_WIDTH=22
export NIDO_TUI_GAP_SCALE=2
export NIDO_TUI_FOOTER_LINK=https://example.com
export NIDO_TUI_TAB_LABELS="1 HOME,2 CREATE,3 LOGS,4 SETTINGS,5 HELP"
```

## What each setting does

- `*_SIDEBAR_*`: widths for sidebars in regular and wide breakpoints.
- `TUI_INSET_CONTENT`: horizontal padding (left+right) for main content containers.
- `TUI_TAB_MIN_WIDTH`: minimum width per tab cell to avoid collapsing on narrow terminals.
- `TUI_EXIT_ZONE_WIDTH`: clickable width for the exit button.
- `TUI_GAP_SCALE`: multiplier applied to spacing tokens (tighten/loosen gaps).
- `TUI_FOOTER_LINK`: URL shown in the footer status bar.
- `TUI_TAB_LABELS`: comma-separated list of 5 labels for the tab bar.

## Defaults (if not overridden)

- Sidebar widths: 18 (regular) / 28 (wide)
- Inset content: 4 (2 cells per side)
- Tab min width: 6
- Exit zone width: 4
- Gap scale: 1
- Footer link: https://github.com/Josepavese
- Tab labels: `1 FLEET,2 HATCHERY,3 LOGS,4 CONFIG,5 HELP`
