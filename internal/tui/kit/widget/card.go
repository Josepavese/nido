package widget

import (
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Card is a static display element with an Icon, Title, and Subtitle.
// It shares the visual style of Form fields but acts as a header/info block.
type Card struct {
	Icon     string
	Title    string
	Subtitle string
	width    int
}

// NewCard creates a new Card widget.
func NewCard(icon, title, subtitle string) *Card {
	return &Card{Icon: icon, Title: title, Subtitle: subtitle}
}

// Interface compliance for Form Element
func (c *Card) Focus() tea.Cmd                        { return nil }
func (c *Card) Blur()                                 {}
func (c *Card) Focused() bool                         { return false }
func (c *Card) Focusable() bool                       { return false }
func (c *Card) Update(msg tea.Msg) (Element, tea.Cmd) { return c, nil }
func (c *Card) SetWidth(w int)                        { c.width = w }

func (c *Card) View(width int) string {
	if width == 0 {
		width = c.width
	}
	style := theme.Current().Styles.Border.Copy().
		Width(width - 2)
		// Inherit(c.Style) // This line is commented out as Card struct does not have a Style field.

	// Calculate Available Space for Text
	// Width - 2 (Border) - 1 (Safety Margin) -> Total Content Width
	// Icon takes constant width (4 via RenderIcon)
	// We need to fit Title and Subtitle in the rest.

	totalContentWidth := width - 3 // -2 border, -1 safety
	if totalContentWidth < 0 {
		totalContentWidth = 0
	}

	iconStr := theme.RenderIcon(c.Icon)
	iconW := lipgloss.Width(iconStr)

	textAvail := totalContentWidth - iconW
	if textAvail < 0 {
		textAvail = 0
	}

	// Prepare Text Components
	// Subtitle has PaddingLeft(2) -> effectively consumes +2 chars
	subRaw := c.Subtitle
	subPadding := 2
	if subRaw == "" {
		subPadding = 0
	}
	subW := lipgloss.Width(subRaw) + subPadding

	titleRaw := c.Title
	titleW := lipgloss.Width(titleRaw)

	// Truncation Logic
	// 1. If Title + Subtitle fits, great.
	// 2. If not, prioritize Title? Or truncate Title to fit Subtitle?
	// Strategy: Truncate Title first (as Subtitle acts as status often).
	// If Title is too long, cut it.

	if titleW+subW > textAvail {
		// Try to shrink Title
		maxTitleW := textAvail - subW
		if maxTitleW < 4 { // If squeezing too much, drop subtitle
			maxTitleW = textAvail
			subRaw = "" // Drop subtitle
			subPadding = 0
		}

		// Truncate Title loop
		runes := []rune(titleRaw)
		for len(runes) > 0 && lipgloss.Width(string(runes)) > maxTitleW {
			runes = runes[:len(runes)-1]
		}
		titleRaw = string(runes)
	}

	// Render Components
	titleStyled := theme.Current().Styles.AccentStrong.Render(titleRaw)

	subStyled := ""
	if subRaw != "" {
		subStyled = lipgloss.NewStyle().Foreground(theme.Current().Palette.TextDim).PaddingLeft(subPadding).Render(subRaw)
	}

	// Join Horizontal (Guaranteed to fit)
	titleParts := lipgloss.JoinHorizontal(lipgloss.Top,
		iconStr,
		titleStyled,
		subStyled,
	)

	return style.Render(titleParts)
}
