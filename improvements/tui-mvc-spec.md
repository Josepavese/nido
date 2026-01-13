# TUI MVC Specification Concept

This document sketches a formal, machine-digestible description for Nido’s TUI, aiming to evolve into a reusable MVC-style framework. The goal: define pages, layout, inputs, navigation, and actions in a structured schema (JSON/YAML) that the framework can render and wire automatically.

## Core Idea
Represent the TUI as data:
- **Model**: domain/state sources (e.g., VM list, cache info) exposed as named data providers.
- **View**: pages composed of components (table, list, form, markdown, status bar) with layout hints and bindings to model fields.
- **Controller**: actions (commands/msgs) bound to events (key, click, selection, submit) with routing rules between pages.

The framework consumes a spec file and produces:
- Rendered pages with consistent theme/layout.
- Key and mouse bindings as declared.
- Wiring between components and actions without handwritten glue.

## Proposed Schema (YAML example)
```yaml
app:
  name: Nido TUI
  theme: auto
  layout:
    breakpoints: {narrow: 100, wide: 140}
    sidebar: {regular: 18, wide: 28}
    inset: 4
    tabMin: 6
pages:
  - id: fleet
    title: Fleet
    layout: split # sidebar + main
    sidebar:
      component: list
      data: vmList
      actions:
        click: selectVM
        key:
          enter: toggleVM
          del: deleteVM
    main:
      component: table
      data: vmTable
      actions:
        key:
          enter: toggleVM
          s: sshVM
          i: infoVM
    navigation:
      tabIndex: 0
      keys: [ "1" ]
  - id: hatchery
    title: Hatchery
    layout: split
    sidebar:
      component: list
      data: hatchModes
      actions: {click: selectMode}
    main:
      component: form
      fields:
        - name: vmName
          label: Name
          validate: vmName
        - name: source
          label: Source
          component: selector
          data: sources
        - name: gui
          label: GUI
          component: toggle
      actions:
        submit: spawnVM
    navigation:
      tabIndex: 1
      keys: [ "2" ]
  - id: logs
    title: Logs
    layout: full
    main:
      component: logviewer
      data: logs
    navigation:
      tabIndex: 2
      keys: [ "3" ]
  - id: config
    title: Config
    layout: split
    sidebar:
      component: list
      data: configItems
      actions: {click: selectConfig}
    main:
      component: form
      fields: dynamicFrom(configSelection)
      actions:
        submit: saveConfig
    navigation:
      tabIndex: 3
      keys: [ "4" ]
  - id: help
    title: Help
    layout: full
    main:
      component: markdown
      data: helpContent
    navigation:
      tabIndex: 4
      keys: [ "5", "h" ]
model:
  providers:
    vmList: {source: service, name: vm.list}
    vmTable: {source: service, name: vm.list}
    logs: {source: service, name: logs.stream}
    configItems: {source: service, name: config.list}
actions:
  selectVM: {type: state, set: {selectedVM: "{{item.name}}"}}
  toggleVM: {type: command, name: vm.toggle, args: {name: "{{selectedVM}}" }}
  deleteVM: {type: command, name: vm.delete, args: {name: "{{selectedVM}}" }}
  infoVM: {type: command, name: vm.info, args: {name: "{{selectedVM}}" }}
  sshVM: {type: command, name: vm.ssh, args: {name: "{{selectedVM}}" }}
  spawnVM: {type: command, name: vm.spawn, argsFromForm: true}
  selectMode: {type: state, set: {mode: "{{item.id}}"}}
  selectConfig: {type: state, set: {configKey: "{{item.key}}"}}
  saveConfig: {type: command, name: config.save, argsFromForm: true}
keymap:
  global:
    quit: ["q", "ctrl+c"]
    nextTab: ["right"]
    prevTab: ["left"]
    refresh: ["r"]
```

## Per-page descriptors + root manifest

- **Root manifest** (`app.yaml`):
  - App metadata (name, theme, layout tokens/overrides, breakpoints, sidebar widths, inset, gap scale).
  - Global keymap and actions.
  - Data providers and validator map.
  - Pages list with paths to page descriptors and tab order.

- **Page descriptor** (e.g., `fleet/page.yaml`):
  - `id`, `title`, `layout` (`split`/`full`).
  - `navigation` (tab index, tab keys, tab label).
  - `sidebar`/`main` components with data bindings and actions.
  - Page-level key overrides (optional).
  - Page-specific strings/icons.

Example root manifest:
```yaml
app:
  name: Nido TUI
  theme: auto
  layout:
    breakpoints: {narrow: 100, wide: 140}
    sidebar: {regular: 18, wide: 28}
    inset: 4
    tabMin: 6
    exitZone: 4
    gapScale: 1
keymap:
  global:
    quit: ["q", "ctrl+c"]
    nextTab: ["right"]
    prevTab: ["left"]
    refresh: ["r"]
pages:
  - tabIndex: 0
    label: "1 FLEET"
    path: fleet/page.yaml
  - tabIndex: 1
    label: "2 HATCHERY"
    path: hatchery/page.yaml
  - tabIndex: 2
    label: "3 LOGS"
    path: logs/page.yaml
  - tabIndex: 3
    label: "4 CONFIG"
    path: config/page.yaml
  - tabIndex: 4
    label: "5 HELP"
    path: help/page.yaml
providers:
  vmList: {source: service, name: vm.list}
  vmTable: {source: service, name: vm.list}
  logs: {source: service, name: logs.stream}
  configItems: {source: service, name: config.list}
validators:
  vmName: vmName
  templateName: vmName
```

Example page descriptor (`fleet/page.yaml`):
```yaml
id: fleet
title: Fleet
layout: split
navigation:
  tabIndex: 0
  keys: ["1"]
  label: "1 FLEET"
sidebar:
  component: list
  data: vmList
  actions:
    click: selectVM
    key:
      enter: toggleVM
      del: deleteVM
main:
  component: table
  data: vmTable
  actions:
    key:
      enter: toggleVM
      s: sshVM
      i: infoVM
```

## Framework loader expectations

- Reads root manifest, then loads each page descriptor.
- Merges global keymap/actions with page-specific ones.
- Validates component/action/provider references.
- Provides theme/layout tokens and override hooks (config/env).
- Renders via Bubble Tea/Bubbles/Lipgloss using the declared components/layouts.

## Benefits
- Single source of truth for pages, layout, navigation, and actions.
- Page isolation: each folder owns its schema/assets, easily swappable.
- Lower drift: rendering and hit-testing can be driven from the same spec and tokens.
- Reuse across projects: swap providers/actions; keep the same page schemas.

## Next Steps
- Lock the schema (YAML/JSON), naming, and required fields.
- Prototype a parser that builds runtime structures (pages, components, actions, keymaps).
- Static render from spec, then add key/mouse routing and data bindings.
- Document component types and bindings contract to keep the schema constrained.
## Framework Expectations
- **Renderer**: interprets pages/layout/components, applies theme tokens, sizes via layout helpers, renders using Bubble Tea/Bubbles/Lipgloss.
- **Event router**: uses declared `navigation`, `actions`, and `keymap` to dispatch events (keys/clicks) to actions or viewlet widget.
- **Data binding**: components bind to providers/state via simple templates (`{{ }}`), with service adapters for commands/queries.
- **Validation**: form fields reference validators by name; framework maps to actual functions.
- **Overrides**: theme/layout/keymap can be overridden via config/env without changing the spec.

## Benefits
- Single source of truth for pages, layout, navigation, and actions.
- Easier reuse across projects: swap providers/actions, keep the same page schema.
- Lower drift: rendering and hit-testing can be derived from the spec and shared layout/theme tokens.

## Next Steps
- Decide on JSON vs YAML (YAML favored for readability).
- Prototype a small parser → runtime state (pages, components, actions).
- Start with static rendering from spec (no actions), then layer in key/mouse routing and bindings.
- Document required component types and their expected bindings/actions to keep the schema constrained and predictable.
