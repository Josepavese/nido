package about

import (
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/build"
	"github.com/Josepavese/nido/internal/tui/kit/layout"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
	view "github.com/Josepavese/nido/internal/tui/kit/view"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	blinkInterval = 800 * time.Millisecond
)

type tickMsg time.Time

// About implements the "About" / "Nido" page in 80s Arcade style.
type About struct {
	view.BaseViewlet
	blinkState bool
}

func NewAbout() *About {
	return &About{
		blinkState: true,
	}
}

func (a *About) Init() tea.Cmd {
	return a.tick()
}

func (a *About) tick() tea.Cmd {
	return tea.Tick(blinkInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *About) Update(msg tea.Msg) (view.Viewlet, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		a.blinkState = !a.blinkState
		return a, a.tick()
	}
	return a, nil
}

func (a *About) View() string {
	t := theme.Current()
	w := a.Width()
	h := a.Height()

	if w == 0 || h == 0 {
		return ""
	}

	// Dynamic Widths based on viewport
	contentWidth := w - 4 // Padding for borders
	if contentWidth > 80 {
		contentWidth = 80
	}

	// --- STYLES ---

	// Arcade Neon Palette
	colorP1 := t.Palette.AccentStrong // Magenta/Pinkish usually
	colorP2 := t.Palette.Accent       // Blue/Cyan usually
	colorCoin := t.Palette.Warning    // Yellow/Gold
	colorText := t.Palette.Success    // Green (Matrix/Retro)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(colorP1).
		Padding(1, 2).
		Width(contentWidth).
		Align(lipgloss.Center)

	// --- COMPACT STYLES ---

	subtitleStyle := lipgloss.NewStyle().
		Foreground(colorP2).
		Bold(true)
		// Removed MarginBottom(1)

	textStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextDim).
		Align(lipgloss.Center)

	highScoreHeaderStyle := lipgloss.NewStyle().
		Foreground(colorCoin).
		Bold(true).
		Underline(true)
		// Removed MarginTop(1)

	// --- COMPONENTS ---

	// 1. HEADER (ASCII ART + BLINKER)
	// EMPIRICAL FIX: Lines must have identical width (24) to prevent drifting during centering.
	// We pad trailing spaces to match the longest line (Line 4).
	logoLines := []string{
		`   _  __ (_) ____  ____  `, // 25
		`  / |/ // / / __ \/ _  \ `, // 25
		` /    // / / /_/ / (_) / `, // 25
		`/_/|_//_/ /____/ \____/  `, // 25
	}
	var styledLines []string
	logoColors := []lipgloss.Style{
		t.Styles.AccentStrong,
		t.Styles.Accent,
		t.Styles.Success,
		t.Styles.Warning,
	}
	for i, line := range logoLines {
		style := logoColors[i%len(logoColors)].Bold(true)
		styledLines = append(styledLines, style.Render(line))
	}

	// Join logo lines with Left alignment to preserve internal structure
	// This creates a stable 24x4 block.
	logoBlock := lipgloss.JoinVertical(lipgloss.Left, styledLines...)

	// Blinking Text
	coinText := "INSERT COIN TO START"
	if !a.blinkState {
		coinText = "                    " // Placeholder to keep layout stable
	}
	blink := lipgloss.NewStyle().Foreground(colorCoin).MarginTop(1).Render(coinText)

	header := lipgloss.JoinVertical(lipgloss.Center,
		logoBlock,
		blink,
	)

	// 2. MISSION BRIEFING (Description)
	missionHeader := subtitleStyle.Render("/// MISSION BRIEFING ///")
	missionText := textStyle.Render(`Nido is a professional, nerdy, and playful VM manager.
While built AI-first, we've graciously included this TUI for our 
human friends—who, due to biological limitations, still require 
primitive visual interfaces and "typing" to feel in control.`)

	mission := lipgloss.JoinVertical(lipgloss.Center,
		missionHeader,
		missionText,
	)

	// 3. HALL OF FLOCK (Credits)
	scoreHeader := highScoreHeaderStyle.Render("HALL OF FLOCK")

	// Table Rows
	rowStyle := lipgloss.NewStyle().Foreground(colorText)
	highlightStyle := lipgloss.NewStyle().Foreground(colorP1).Bold(true)

	// Creator Link
	creatorName := "\x1b]8;;https://github.com/Josepavese\x1b\\JOSÉ PAVESE\x1b]8;;\x1b\\"

	// Helper to format a score row
	makeRow := func(rank, name, score string, isHighlight bool) string {
		s := rowStyle
		if isHighlight {
			s = highlightStyle
		}
		// Manually spacing for "Table" look without complex table widget overhead for 3 lines
		// Rank: 5 chars, Name: 20 chars, Score: 10 chars
		return s.Render(padRight(rank, 5) + padRight(name, 20) + padLeft(score, 10))
	}

	scores := lipgloss.JoinVertical(lipgloss.Center,
		makeRow("1ST", creatorName, "999,999", true),
		makeRow("2ND", "NIDO AI", "900,000", false),
		makeRow("3RD", "YOUR AGENT", "000,000", false),
	)

	credits := lipgloss.JoinVertical(lipgloss.Center,
		scoreHeader,
		scores,
		t.Styles.TextMuted.Render("https://github.com/Josepavese/nido"),
	)

	// 4. FOOTER (Version)
	ver := t.Styles.TextMuted.Render("VER " + build.Version + " [EXP]")

	// --- ASSEMBLY ---
	// Join all parts vertically - COMPRESSED (Minimal vertical gaps)
	innerContent := lipgloss.JoinVertical(lipgloss.Center,
		header,
		mission,
		credits,
		ver,
	)

	// Wrap in borders and place in center of screen
	bo := borderStyle.Render(innerContent)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, bo)
}

func (a *About) Resize(rect layout.Rect) {
	a.BaseViewlet.Resize(rect)
}

func (a *About) HandleMouse(x, y int, msg tea.MouseMsg) (view.Viewlet, tea.Cmd, bool) {
	return a, nil, false
}

func (a *About) Shortcuts() []view.Shortcut {
	return nil
}

func (a *About) IsModalActive() bool {
	return false
}

func (a *About) HasActiveTextInput() bool {
	return false
}

func (a *About) HasActiveFocus() bool {
	return false
}

// Helpters for table alignment
func padRight(s string, width int) string {
	l := lipgloss.Width(s)
	if l < width {
		return s + strings.Repeat(" ", width-l)
	}
	return s
}

func padLeft(s string, width int) string {
	l := lipgloss.Width(s)
	if l < width {
		return strings.Repeat(" ", width-l) + s
	}
	return s
}
