# Nido TUI Kit

The Nido TUI Kit is a reusable, component-based library for building terminal user interfaces with Bubble Tea and Lip Gloss. It provides a set of high-level widgets and layout utilities designed for consistent styling and ease of use.

## Widgets

### Card

A static display element with an Icon, Title, and Subtitle. Used for headers or info blocks.

* **Usage**: `widget.NewCard("Icon", "Title", "Subtitle")`
* **Style**: Matches the visual footprint of Form fields (Boxed).

### Declarative Form

The `widget.Form` allows you to build complex forms declaratively using a list of elements. It handles focus cycling (Tab/Shift+Tab), validation, and layout automatically.

**Features:**

* **Declarative Syntax**: Compose forms using `NewForm(ele1, ele2, ...)`.
* **Focus Management**: Automagical cycling with `NextField()` / `PrevField()`.
* **Consistent Styling**: All fields use `RenderBoxedField` for a unified "Boxed" aesthetic with left-aligned labels.
* **Validation**: Built-in visual feedback for errors.
* **Disabled State**: Elements can be marked `Disabled: true` to become read-only and skipped during focus cycling.
* **Compact Mode**: Elements can be marked `Compact: true` for single-line rendering without borders (opt-in).
* **Smart Submit**: Buttons created with `NewSubmitButton` are automatically skipped (unfocusable) if the form contains validation errors.
* **Real-time Filtering**: Inputs support a `Filter func(rune) bool` to block invalid characters before they are even entered.
* **Adaptive Rendering**: Inputs automatically adapt to tight spaces (shorter labels, content truncation with "...").

**Elements:**

* **`Card`**: A non-interactive header/info block.

    ```go
    widget.NewCard("Icon", "Title", "Subtitle")
    ```

* **`Input`**: A text input field with validation.

    ```go
    widget.NewInput("Label", "Placeholder", validatorFunc)
    // Supports i.Disabled = true, i.Compact = true, i.Filter = filterFunc
    ```

* **`Toggle`**: A binary on/off switch.

    ```go
    widget.NewToggle("Label", initialValue)
    // Supports t.Disabled = true
    ```

* **`Button`**: An action button.
  * `Centered: true`: Centers the button text and removes the label (useful for inline actions).

    ```go
    widget.NewButton("Label", "Button Text", actionFunc)
    // Or for invalid-blocking behavior:
    widget.NewSubmitButton("Label", "Submit", actionFunc)
    ```

* **`Row`**: Horizontal layout container for elements. Implements `Navigator` interface to support seamless `Tab` cycling through its children.

    ```go
    // Equal width distribution
    row := widget.NewRow(input1, input2, button)
    
    // Proportional width distribution (e.g., 2/3 and 1/3)
    row := widget.NewRowWithWeights(
        []widget.Element{diskInput, deleteButton},
        []int{2, 1}, // diskInput gets 2/3, deleteButton gets 1/3
    )
    ```

### Modal

A blocking overlay dialog for confirmations or alerts.

* **Usage**: `widget.NewModal(title, msg, onConfirm, onCancel)`
* **Behavior**:
  * `Show()` / `Hide()` visibility control.
  * Overlay logic handled by rendering `View(w,h)` on top of parent content.
  * Traps input (`Update`) when active.
* **Example**:

    ```go
    modal := widget.NewModal("Confirm", "Are you sure?", doAction, cancelAction)
    if modal.IsActive() { return modal.View(width, height) }
    ```

### SidebarList

A scrollable list component optimized for navigation sidebars.

* Items implement `SidebarItem.SelectionID()` with a stable domain identity,
  separate from labels, spinner frames and other transient presentation state.
* `SetItems` preserves the highlighted identity across refresh/reorder and returns
  false when a selected item disappears. Pages retain their own removal policy.
* `Select` returns a notification command. Keyboard and mouse use the same path.
  Handle `widget.SelectionMsg` only through `Sidebar.CurrentSelection(msg)`;
  it rejects superseded events and events from other lists on the UI loop.
* Selection dialogs share identity restoration through `ListModal.SetItems`.
  Do not replace a dialog's embedded Bubbles items directly.
* Customizable icons and styling.
* Handles selection state and filtering (search).

### MasterDetail

A layout container that pairs a `SidebarList` (Master) with a content view (Detail).

* Responsive resizing logic.
* Manages focus switching between Sidebar and Detail pane.
* Automatically selects first item when list is populated.

### PageManager

A simple stack-based page switcher.

* `AddPage(id, viewlet)`
* `SwitchTo(id)`

## Usage Example

```go
// Create Elements
card := widget.NewCard("🧬", "Configuration", "v1.0")
nameInput := widget.NewInput("Name", "Enter value...", nil)
toggle := widget.NewToggle("Advanced Mode", false)

// Create Row with proportional widths
row := widget.NewRowWithWeights(
    []widget.Element{nameInput, toggle},
    []int{3, 1}, // nameInput gets 3/4, toggle gets 1/4
)

// Create Form
form := widget.NewForm(card, row)

// Update Loop
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Delegate to form
    newForm, cmd := m.form.Update(msg)
    m.form = newForm
    return m, cmd
}

// View
func (m Model) View() string {
    return m.form.Viewport(width, height)
}
```

## Responsive contract

The kit owns terminal geometry (`layout`), theme tokens, reflow and hit testing.
Pages pass their measured rectangle to `Form.Viewport(layout.FormWidth(...), height)`.
Do not impose a local minimum width or draw a second version of a form row.

- `SplitView` shows one pane when a sidebar and a useful detail no longer fit.
  `MasterDetail` owns the visible pane, Enter/Tab to open and Esc to return.
  A resize preserves selection, focus and draft values.
- `Row` reflows into stacked fields when its weighted columns become too small.
  Drawing, mouse hits and focus following consume the same rectangles.
- `ScrollView` keeps a changed focus visible and supports manual scrolling without
  snapping back. Forms accept PgUp/PgDn and the mouse wheel, including read-only
  details. Its last line shows the current range when content overflows.
- `TabBar` returns content and click rectangles together. The compact bar retains
  all routes. `StatusBar` keeps complete hints and the exit command, prioritizing
  the current task. Input arrows move the caret; Ctrl+Left/Right changes routes.
- All overlays use the same measured dialog frame. Confirmation controls remain
  visible while long messages scroll. Cancel remains the default selection.
- Below the viable terminal dimensions, `RenderResizeNotice` rests the interface
  and suspends hidden input. Background messages continue; Ctrl+C still exits.
  Zero-sized terminals render nothing. The size policy lives in `layout`.

Every TUI change requires visual inspection of actual terminal output at wide,
compact and minimum dimensions, including a shrink/expand round trip, drafts,
long/Unicode content and active overlays. Unit geometry checks do not replace
this inspection. The native PTY journey and optional HTML observer are documented
in [TESTING.md](../../../TESTING.md#tui-resize-and-visual-inspection).

### Empty states, selections and overlays

`RenderEmptyState(icon, title, message, width, height)` is the shared invitation
for unselected details. Pages supply domain wording and theme icons; the kit
measures, wraps and places it. Cards balance their title with secondary text
using cell-safe truncation. Button rows use the buttons' actual content budget
before stacking them.

`Overlay` includes its contextual `Shortcuts()`. `ActiveOverlay` resolves a
page's priority once for drawing, mouse input and hints. `Shell.Overlay` hosts
application-wide dialogs through the same body geometry and footer. Modal input
blocks route changes while Ctrl+C/Ctrl+Q remain available. Footers deduplicate
keys; confirmation defaults to cancellation and normal alerts carry an explicit
severity. A form modal uses Tab to reach its actions and Esc to cancel.

List viewports own a single numeric pagination row, without the underlying
list's extra spacer. Items may implement `Selectable() bool` for visible,
informational sections that keyboard and mouse navigation skip. List dialogs
shrink to their content and retain their preferred dimensions as upper bounds.
Theme names have a stable order; live preview is a draft until saved, and
cancelling the picker restores the selected theme.
