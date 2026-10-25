package widget

import (
	"github.com/Josepavese/nido/v5/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v5/internal/tui/kit/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	if width <= 0 {
		return ""
	}
	t := theme.Current()
	icon := theme.RenderIcon(c.Icon)
	available := max(0, width-layout.DialogBorder-lipgloss.Width(icon))
	titleBudget := available
	gap := theme.Space.SM
	if c.Subtitle != "" {
		titleBudget = max(available/2, available-lipgloss.Width(c.Subtitle)-gap)
	}
	title := ansi.Truncate(c.Title, titleBudget, "…")
	subtitle := ansi.Truncate(c.Subtitle, max(0, available-lipgloss.Width(title)-gap), "…")
	content := icon + t.Styles.AccentStrong.Render(title)
	if subtitle != "" {
		content += t.Styles.TextDim.PaddingLeft(gap).Render(subtitle)
	}
	box := t.Styles.Border.Width(max(0, width-layout.DialogBorder)).Render(content)
	return layout.Fit(width, lipgloss.Height(box), box)
}
