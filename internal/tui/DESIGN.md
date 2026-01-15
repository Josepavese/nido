# Nido TUI Design Specification

> **Status**: APPROVED  
> **Date**: 2026-01-14  
> **Goal**: To transform the Nido TUI into a "Superb" aesthetic and highly efficient command center.

## 1. Executive Summary

The Nido TUI will evolve from a simple list view into a **Rich Command Center**. It will provide full parity with the CLI while offering a superior UX through visualization, rapid keyboard navigation, and intuitive mouse interactions. The aesthetic will be "Premium Terminal" — using modern TUI components, subtle gradients/colors, and clear hierarchy.

## 2. Design Principles

1. **Efficiency First**: Every action available via CLI must be faster or equally fast in TUI.
2. **Hybrid Interaction**:
    - **Keyboard Power usage**: `j/k` navigation, `Enter` to act, `Tab` to cycle.
    - **Mouse Friendliness**: Everything clickable, scrollable.
3. **Visual Excellence**:
    - Use "Cockpit" layouts (sidebar lists + rich detail panels).
    - Status indicators (dots, badges) must be instantly readable.
    - Zero interference: Minimal chrome, maximum content.

## 3. Architecture & Navigation

### Global Layout

The Shell follows a standard "Application Frame":

- **Header**: Minimal tabs (Icons + Labels). Clickable.
- **Body**: The active Page (Viewlet).
- **Footer**: Contextual shortcuts key map.

### Routing / Pages

1. **🦅 FLEET** (The Cockpit) - *Default*
2. **🐣 HATCHERY** (The Creator)
3. **💿 REGISTRY** (The Library) - *Merges Images & Cache*
4. **⚙️ SYSTEM** (The Engine) - *Config, Doctor, Update*

## 4. Interaction Model

### 🖱️ Mouse capabilities

The entire UI is "Touch-First" compatible:

- **Navigation**: Click Header Tabs to switch pages.
- **Lists**: Click any Sidebar item to select it. Scroll wheel works on all lists.
- **Buttons**: All `[ BUTTON ]` elements are clickable.
- **text Inputs**: Click to focus (where applicable).
- **Copy/Paste**: Critical data (IPs, SSH strings) must be rendered in standard text for OS selection.

### ⌨️ Footer (Contextual Help)

The footer dynamically updates based on the active Viewlet:

- **Global**: `Tab` (Next Page), `Ctrl+C` (Exit).
- **Global**: `Tab` (Next Page), `Ctrl+C` (Exit).
- **Fleet**: `↑/↓` (Nav), `Enter` (Power Toggle), `s` (SSH), `v` (VNC), `x` (Stop), `Del` (Destroy).
- **Hatchery**: `Tab` (Next Field), `Space` (Select), `Enter` (Hatch).
- **Registry**: `Tab` (Switch Remote/Local), `p` (Pull/Prune).

## 5. Page Detailed Design

### 5.1. 🦅 FLEET (The Cockpit)

**Status**: Implemented ✅
**Layout**: Master-Detail

- **Left (Sidebar)**:
  - Scrollable List of VMs.
  - Item: `[StatusDot] [Name] [State]`
  - Behavior: Auto-refreshes every 2s.
- **Right (Detail)**: "The Cockpit" representing the VM state.
  - **Header Card**: Name, large status icon/text.
  - **Identity/Network**: Read-only Form fields for PID, IP.
  - **Connectivity**: Read-only Form fields for SSH/VNC ports.
  - **Storage**: Read-only Form field for Disk path (Red border if missing).
  - **Actions (Shortcuts)**:
    - `Space`: Toggle Power (Start/Stop).
    - `s`: Open SSH.
    - `v`: Open VNC.
    - `Del`: Destroy VM (triggers Confirmation Modal).

### 5.2. 🐣 HATCHERY (The Creator)

**Status**: Implemented ✅
**Layout**: Master-Detail (Wizard)

- **Left (Sidebar)**:
  - Inventory of "Sources" (Templates `🧬` & Cloud Images `💿`).
  - Item: `[Icon] [Name]`
  - Behavior: Selects source context for Incubator.
- **Right (Incubator)**: Configuration Form.
  - **Header Card**: Selected Source info.
  - **Form Fields**:
    - Name (Input, auto-generated default).
    - GUI Mode (Toggle).
    - *Future*: CPU/RAM/Disk size.
  - **Actions**:
    - `Enter`: Hatch (Spawn VM).
    - `Del`: Delete selected Template (triggers Modal).

### 5.3. 💿 REGISTRY (The Library)

**Layout**: Tabbed List or Master-Detail

- **Tabs**: "Remote Catalog" vs "Local Cache".
  - *Mouse*: Click to switch.
- **Remote**: Browse official images. Action: `[ PULL ]`.
- **Local**: List cached images.
  - Action: `[ PRUNE CACHE ]` (Delete all unused).
  - Action: `[ DELETE ]` (Delete specific image).
- **Visuals**: Show download size, age, distribution logos (icons).

### 5.4. ⚙️ SYSTEM (The Engine)

**Layout**: Dashboard Grid

- **Version**: Current ver + "Check Update" button -> `[ UPGRADE ]` if available.
- **Doctor**: Run Health Check output log.
- **Config**: List of Key/Values.
  - Action: `[ EDIT ]` button to modify values (e.g., `SSH_USER`).

## 6. Technical Standardization

- **Framework**: Bubble Tea + Lip Gloss.
- **Components**: Reuse `MasterDetail`, `SidebarList`, `Table` widgets.
- **Theme**: Centralized `theme` package for colors (Success=Green, Error=Red, Accent=Blue/Purple).
- **Wiring**: `ops` package handles all business logic (wrapping provider).

## 7. Implementation Phases

1. **Refine Fleet**: Polish the "Cockpit" (Complete).
2. **Revamp Hatchery**: Move to Wizard layout.
3. **Build Registry**: Consolidate Images/Cache.
4. **Build System**: Dashboard for miscellaneous ops.
