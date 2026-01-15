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
* **Adaptive Rendering**: Inputs automatically adapt to tight spaces (shorter labels, content truncation with "...").

**Elements:**

* **`Card`**: A non-interactive header/info block.

    ```go
    widget.NewCard("Icon", "Title", "Subtitle")
    ```

* **`Input`**: A text input field with validation.

    ```go
    widget.NewInput("Label", "Placeholder", validatorFunc)
    // Supports i.Disabled = true, i.Compact = true
    ```

* **`Toggle`**: A binary on/off switch.

    ```go
    widget.NewToggle("Label", initialValue)
    // Supports t.Disabled = true
    ```

* **`Button`**: An action button.

    ```go
    widget.NewButton("Label", "Button Text", actionFunc)
    // Or for invalid-blocking behavior:
    widget.NewSubmitButton("Label", "Submit", actionFunc)
    ```

* **`Row`**: Horizontal layout container for elements.

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

* Supports generic items via `SidebarItem` interface.
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
    return m.form.View()
}
```
