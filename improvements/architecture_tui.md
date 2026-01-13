# Nido TUI Architecture (Native Go Standard)

**Status**: ACTIVE
**Type**: Architecture Reference
**Date**: 2026-01-13
**Version**: 1.0.0

## 1. Executive Summary

The Nido TUI (Terminal User Interface) is a high-performance, low-latency control plane for managing local virtual machines. It is built on the **Bubble Tea** framework but employs a strict **Native Go MVC** architecture to separate the generic UI engine (`kit`) from specific application logic (`app`). This ensures that the core framework remains reusable and testable ("Write Once, Play Multiple"), while the application layer remains focused purely on business rules and data.

## 2. Scope & Non-Goals

### In-Scope

* **The Kit**: A generic TUI browser engine (Shell, Layout, Widgets).
* **The App**: Nido-specific implementation (Pages, Wiring, Providers).
* **MVC Pattern**: Strict enforcing of Model (Data), View (Render), Controller (Update) via Go interfaces.

### Out-of-Scope

* **YAML Runtime**: We explicitly reject defining UIs via external configuration files (e.g., YAML/JSON). The code *is* the schema.
* **Web Rendering**: This architecture controls the Terminal *only*.
* **Remote Management**: The TUI is designed for local-first interaction.

## 3. System Overview

The system is bisected into two primary domains:

1. **`internal/tui/kit` (The Framework)**:
    * Acts as the "Operating System" for the TUI.
    * Manages the Event Loop (`Init`, `Update`, `View`).
    * Handles Window Management (Layout, Resizing, Chrome).
    * Provides primitives (Tables, Forms, Lists).
    * **Constraint**: Must not import `internal/tui/app`.

2. **`internal/tui/app` (The Application)**:
    * Acts as the "User Space" program.
    * Implements specific Pages (`Fleet`, `Hatchery`).
    * Injects Data Providers (`VMProvider`).
    * Wires the system together.

## 4. Component Architecture

### 4.1. The Generic App (`kit/app`)

* **`App`**: The root model that satisfies the `tea.Model` interface.
* **`Shell`**: The visual container. Manages the Header, Footer, and the Active Viewlet. It is responsible for **Chrome Rendering** (Tabs, Spinners, Toasts).

### 4.2. The Viewlet (`kit/view`)

The fundamental unit of the UI is the `Viewlet` interface:

```go
type Viewlet interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Viewlet, tea.Cmd)
    View() string
    Resize(r layout.Rect)
    HandleMouse(x, y int, msg tea.MouseMsg) (Viewlet, tea.Cmd, handled bool)
    Shortcuts() []Shortcut
}
```

* **Autonomy**: Viewlets handle their own input and rendering logic.
* **Composition**: Viewlets can contain other Viewlets (e.g., `MasterDetail` contains `SidebarList` and `PageManager`).

### 4.3. The Page Pattern (`app/pages`)

Specific screens are implemented as structs embedding `viewlet.BaseViewlet`:

* **Model**: Holds local UI state (selection, form inputs).
* **Controller**: The `Update()` method receives global messages (e.g., `SelectionMsg`) and specific commands. It delegates business logic to the `VMProvider`.
* **View**: Uses `kit/layout` and `kit/theme` to render strictly sized TUI strings.

## 5. Data Architecture

### 5.1. Providers ("The Backend")

The TUI does not own the data. It queries authoritative sources via interfaces:

* `VMProvider`: Source of truth for running VMs and Templates.
* `ConfigService`: Source of truth for settings.

### 5.2. State Flow (Unidirectional)

1. **Event**: User presses `Refresh` (or Ticker ticks).
2. **Command**: `NidoApp` sends a `Cmd` to fetch data.
3. **Message**: Data returns as a `Msg` (e.g., `FleetMsg`).
4. **Update**: The Active Viewlet receives the `Msg` and updates its internal Model.
5. **View**: The Viewlet re-renders based on the new Model.

## 6. Integration & Interfaces

### 6.1. Wiring (`app/wiring.go`)

The entry point (`Run`) constructs the dependency graph:

1. Initialized the **Theme**.
2. Creates the **Kit App**.
3. Configures the **Shell** (Tabs, Title).
4. Registers **Pages** via `PageManager`.
5. Wraps it all in **`NidoApp`**.

### 6.2. The Nido Adapter (`app.NidoApp`)

A wrapper that sits between Bubble Tea and `kit.App`.

* **Job**: Global Key Handling (Global Quit).
* **Job**: Dependency Injection (Holds `VMProvider`).
* **Job**: Message Interception (Can override behavior before it reaches the Kit).

## 7. Scalability & Performance

* **Mouse Handling**: Centralized in `Shell`. O(1) routing complexity.
* **Resizing**: Propagated via `Resize(Rect)` down the tree. Layouts are strictly calculated, avoiding "float" behavior.
* **Adding Pages**: O(1) complexity. Define a struct, register it in wiring. No other files touch.

## 8. Development Standards

### "Code IS The Schema"

* **Strong Typing**: Use Go structs for everything.
* **Compile-Time Safety**: Interfaces ensure Viewlets implement all required methods.
* **No Magic Strings**: Use constants from `kit/theme` and `kit/view`.

### Naming Conventions

* **Packages**: `kit` components in specific packages (`widget`, `layout`). App pages in `app/pages/<name>`.
* **Messages**: Suffix with `Msg` (e.g., `SelectionMsg`).
* **Commands**: Suffix with `Cmd` (e.g., `LoadVMCmd`).

## 9. Future Roadmap

1. **Form Validation**: Move `kit/validator` usage into a declarative `FormBuilder` widget.
2. **Modal Manager**: Extract Modal logic from `MasterDetail` into a global `OverlayManager` in the Shell.
3. **Test Coverage**: Expand `kit` unit tests to 90% (currently ~60%).

---
*Signed,*
*The Chief Architect*
