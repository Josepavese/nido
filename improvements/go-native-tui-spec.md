# TUI Architecture: The "Native Go" Standard

**Philosophy**: "The Code IS the Schema."

Instead of building a complex YAML interpreter (which adds runtime fragility and huge complexity), we formally standardize the **Go-Native MVC Architecture** we have just built. We rely on Go's compiler for safety and `gopls` for developer experience (autocompletion), rather than validting separate YAML files.

## 1. The Core Split: `kit` vs `app`

We formalize the strict boundary between the Generic Framework and the Application.

### The Kit (`internal/tui/kit`)

* **Role**: The "Browser Engine". Knows *nothing* about VMs, Nido, or business logic.
* **Responsibility**:
  * Rendering the Shell (Chrome, Tabs, Status Bar).
  * Routing events (Key/Mouse) to the active Viewlet.
  * Providing standard widgets (Table, List, Form).
  * **NO** business logic imports.

### The App (`internal/tui/app`)

* **Role**: The "Website". Contains all business logic and specific screens.
* **Responsibility**:
  * **Pages**: Implementation of specific screens (`Fleet`, `Hatchery`).
  * **Wiring**: The glue that initializes the Kit and injects dependencies.
  * **Adapter**: `NidoApp` wraps `kit.App` to bridge custom events (e.g., `RequestSpawnMsg` -> `VMProvider`).

## 2. The MVC Pattern

We strictly enforce MVC using Go Types:

* **Model**: The `Provider` interfaces (e.g., `VMProvider`). The TUI never holds authoritative state; it queries the provider.
* **View**: The `Viewlet` interface.

    ```go
    type Viewlet interface {
        Init() tea.Cmd
        Update(msg tea.Msg) (Viewlet, tea.Cmd)
        View() string
        Resize(r layout.Rect)
        // input
        HandleMouse(x, y int, msg tea.MouseMsg) (Viewlet, tea.Cmd, handled bool)
        Shortcuts() []Shortcut
    }
    ```

* **Controller**: The Page Structs (e.g., `pages.Fleet`). They handle user intent (`enter` key) and translate it into Commands (`vm.Start(id)`).

## 3. Developer Workflow ("Add a Page")

Instead of editing a big YAML, the developer follows a typed path:

1. **Create Struct**: `type MyPage struct { viewlet.BaseViewlet ... }`
2. **Implement View**: Use `kit/layout` helpers.
3. **Register**: Add one line to `app/wiring.go`:

    ```go
    pages.AddPage("MY_PAGE", &MyPage{...})
    ```

## 4. Documentation Plan

We will produce `docs/TUI_ARCHITECTURE.md` covering:

1. **The Component Diagram**: How Shell, PageManager, and Viewlets nest.
2. **The Event Loop**: How a keypress propagates (Shell -> PageManager -> ActivePage -> Widget).
3. **Style Guide**: How to use `kit/theme` tokens correctly.

## 5. Future "Magic" (Optional)

If we want to speed up development later, we write a **Scaffolder**, not a Runtime.
`nido dev gen page my-page` -> Generates the Go struct boilerplate.

* **Benefit**: Result is simple Go code.
* **Cost**: Zero runtime overhead.
