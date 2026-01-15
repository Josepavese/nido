package main

import (
	"fmt"
	"strings"

	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
)

// Mocking the layout logic from Shell and Pages to measure widths perfectly.

func main() {
	// 1. Simulate Terminal Widths
	widths := []int{80, 100, 120, 160}

	for _, w := range widths {
		fmt.Printf("\n--- Testing Terminal Width: %d ---\n", w)
		testLayout(w, 40) // Arbitrary height
	}
}

func testLayout(termW, termH int) {
	// A. Sidebar Calculation (from kit/layout/grid.go & widget/split_view.go)
	// layout.Calculate(r.Width, r.Height, 25, theme.Width.Sidebar, theme.Width.SidebarWide)
	dim := layout.Calculate(termW, termH, 25, theme.Width.Sidebar, theme.Width.SidebarWide)
	sidebarW := dim.SidebarWidth

	// Separator Width (from SplitView)
	separatorW := 1

	// B. Detail View Available Width
	// mainW := r.Width - sidebarWidth - borderW
	availMainW := termW - sidebarW - separatorW
	if availMainW < 0 {
		availMainW = 0
	}

	// C. Registry/Form Logic (from registry.go)
	// padding := theme.Current().Layout.ContainerPadding
	// safeWidth := w - (2 * padding)
	// if safeWidth > 60 { safeWidth = 60 }
	// if safeWidth < 40 { safeWidth = 40 }

	// Assuming ContainerPadding is 2 (need to verify in theme, but using mock for now or importing)
	// Using the actual theme import
	padding := theme.Current().Layout.ContainerPadding // usually 2

	// The View() receives availMainW as width
	detailFormSafeW := availMainW - (2 * padding)

	// Apply Constraints
	if detailFormSafeW > 60 {
		detailFormSafeW = 60
	}
	if detailFormSafeW < 40 {
		detailFormSafeW = 40
	}

	// D. Visual Width Calculation
	// What makes up the "Right Column" visually?
	// It's the form + the padding around it?
	// The Action Stacks aligns left with Sidebar.
	// It should align right with the Form's right edge.

	// Left Edge = 0
	// Sidebar Right Edge = sidebarW
	// Detail Content Start = sidebarW + separatorW + padding
	// Detail Content End = Detail Content Start + detailFormSafeW

	// totalContentWidth := sidebarW + separatorW + padding + detailFormSafeW + padding
	// Unused for now
	// "tra colonna sx + colonna dx"
	// The header card in the screenshot looks like it covers the form width.
	// If the stack is wider, it probably covers the padding + extra.

	// My previous logic in Shell.go was:
	// targetW := sidebarW + 1 + visualDetailW + 1
	// visualDetailW := detailW + 4 (padding*2)

	fmt.Printf("Sidebar:   %d\n", sidebarW)
	fmt.Printf("Separator: %d\n", separatorW)
	fmt.Printf("AvailMain: %d\n", availMainW)
	fmt.Printf("Padding:   %d\n", padding)
	fmt.Printf("FormSafe:  %d (Clamped)\n", detailFormSafeW)

	// Target Calculation
	// We want the Stack to be exactly Sidebar + Separator + Padding + FormWidth + (Maybe Right Padding?)
	// If the stack is a "Card", it has its own borders/padding.
	// widget.ActionStack.View(width) creates a box of `width`.
	// Ideally `width` should equal visible ink of Sidebar + ... + Form.

	// Let's try to match exactly the right edge of the form.
	// Width = (Sidebar) + (Sep) + (LeftPad) + (Form)
	// Why LeftPad? Because form is centered/padded inside the main view.
	// Actually Registry View -> MasterDetail -> Pages -> DetailView -> Form
	// RegistryDetail.View() manually pads? No, it calculates safeWidth and passes it to Form.View(safeWidth).
	// But Form.View returns a string rendered with that width.
	// DOES IT PAD IT?

	// Let's assume standard left alignment in DetailView:
	// return lipgloss.JoinVertical(lipgloss.Left, d.Form.View(safeWidth))
	// Wait, RegistryDetail.View() logic:
	// return d.Form.View(safeWidth)
	// It returns the form string.
	// Who adds the padding to the DetailView?
	// The DetailView is rendered by MasterDetail inside `mainView`.
	// SplitView renders `mainView`.
	// Does SplitView add padding?
	// SplitView line 88: `mainView` (string).
	// It joins: Sidebar + Separator + MainView.

	// So if RegistryDetail just returns `Form.View(safeWidth)`, where is the padding?
	// Ah, `NewRegistryDetail` doesn't seem to wrap it in a padded box.
	// But `RegistryDetail.View` *calculates* width based on padding, but does nothing with it if it doesn't render container styles.
	// If `Form.View` creates a block of `safeWidth`, and `RegistryDetail.View` returns it directly...
	// Then it is rendered immediately after the separator.
	// UNLESS Form.View has margin?
	// Or `MasterDetail`?
	// Wait, `RegistryDetail.View` logic:
	// 411: return d.Form.View(safeWidth)

	// If I modify RegistryDetail to NOT apply padding...
	// The code subtracts padding from width to get safeWidth, but doesn't wrap the result in padding style.
	// So `Form` is rendered with `safeWidth`... but is it indented?
	// If not indented, it touches the separator.
	// Let's look at the screenshot. There is a gap between the vertical line and the "cirros" card.
	// That gap is the padding.
	// If the code doesn't explicitly render padding, where does it come from?
	// Maybe `Form` adds margin?

	// Let's assume for a moment the 'padding' variable is used for sizing but not used for rendering indentation?
	// That would be a bug or I missed something in `Form.View`.

	// HYPOTHESIS: The form is centered or aligned inside the available space?
	// No, `lipgloss.JoinVertical(lipgloss.Left...)` in `Form.View`.

	// Let's verify `Form.View` or assume for now we need to calculate:
	// Width = SidebarW + 1 + (Effective Detail Width)

	// If the UI shows a gap, we must account for it.
	// Let's propose a "Visual Width":
	// StackWidth = SidebarW + 1 + (Gap?) + FormW

	// Let's just output the "Target Width" that covers Sidebar + 1 + Padding + Form
	targetW := sidebarW + separatorW + padding + detailFormSafeW

	fmt.Printf("Calculated Target Width: %d\n", targetW)

	// Visualize
	sb := strings.Repeat("S", sidebarW)
	sp := "|"
	// Simulate padding gap if it exists?
	// If RegistryDetail doesn't add padding, maybe it's 0 visually?
	// But screenshot shows gap.
	// Maybe `theme.Layout.ContainerPadding` is applied inside `Form`?
	// If `Form` uses `width`, does it fill it?

	// Let's construct a visual representation string
	gap := strings.Repeat(" ", padding)
	fm := strings.Repeat("F", detailFormSafeW)

	// This represents what we think the main row looks like:
	// SSSS|  FFFFFF
	virtualRow := sb + sp + gap + fm
	fmt.Printf("Visual Row Len: %d\n", len(virtualRow))

	// The Action Stack (A) should match this length.
	fmt.Printf("Action Stack W: %d\n", len(virtualRow))
}
