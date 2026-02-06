package widget

import (
	"fmt"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/tui/kit/theme"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Action represents a single running operation
type Action struct {
	ID        string
	Message   string
	Progress  float64 // 0.0 to 1.0
	Spinner   spinner.Model
	Bar       progress.Model
	StartTime time.Time
}

// ActionStack manages a stack of running operations
type ActionStack struct {
	Items []Action
}

// NewActionStack creates a new stack
func NewActionStack() *ActionStack {
	return &ActionStack{
		Items: make([]Action, 0),
	}
}

// Init initializes the component
func (s *ActionStack) Init() tea.Cmd {
	return nil
}

// Add starts a new action and returns its ID
func (s *ActionStack) Add(id, message string) tea.Cmd {
	// Deduplicate: if ID exists, update it
	for i, a := range s.Items {
		if a.ID == id {
			s.Items[i].Message = message
			s.Items[i].StartTime = time.Now()
			return nil
		}
	}

	t := theme.Current()

	// Initialize Spinner
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(t.Palette.Accent)

	// Initialize Progress Bar
	pg := progress.New(progress.WithGradient(
		t.Palette.Accent.Dark,
		t.Palette.Success.Dark,
	))
	pg.ShowPercentage = true
	pg.PercentageStyle = t.Styles.Accent // Use Accent color for percentage to match theme

	s.Items = append(s.Items, Action{
		ID:        id,
		Message:   message,
		Progress:  0,
		Spinner:   sp,
		Bar:       pg,
		StartTime: time.Now(),
	})

	return sp.Tick
}

// Update handles animation ticks
func (s *ActionStack) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range s.Items {
		var cmd tea.Cmd
		s.Items[i].Spinner, cmd = s.Items[i].Spinner.Update(msg)
		cmds = append(cmds, cmd)
	}
	// Note: Progress bar Update is mostly for resizing, which we handle in View
	return tea.Batch(cmds...)
}

// UpdateProgress updates the progress of an existing action
func (s *ActionStack) UpdateProgress(id string, p float64) {
	for i, a := range s.Items {
		if a.ID == id {
			s.Items[i].Progress = p
			return
		}
	}
}

// Remove finishes an action
func (s *ActionStack) Remove(id string) {
	newItems := make([]Action, 0)
	for _, a := range s.Items {
		if a.ID != id {
			newItems = append(newItems, a)
		}
	}
	s.Items = newItems
}

// Height returns the number of lines needed to render the stack
func (s *ActionStack) Height() int {
	if len(s.Items) == 0 {
		return 0
	}
	// Compact Card: 1 line of content + 2 lines of border = 3 lines.
	return len(s.Items) * 3
}

// View renders the stack
func (s *ActionStack) View(width int) string {
	if len(s.Items) == 0 {
		return ""
	}

	var rows []string
	t := theme.Current()

	// Card Style (Compact: 1 line of content)
	cardStyle := t.Styles.Border.
		Width(width-2).
		Padding(0, 1)

	for _, a := range s.Items {
		// Prepare Title components
		// IMPORTANT: Style the message SEPARATELY from the spinner.
		// Spinner.View() contains ANSI reset codes that kill any wrapper style.
		spinnerView := a.Spinner.View()
		messageView := t.Styles.AccentStrong.Render(strings.ToUpper(a.Message))

		// 2. Progress Bar
		val := a.Progress
		if val > 1.0 {
			val = 1.0
		}
		if val < 0.0 {
			val = 0.0
		}

		// Calculate available width
		availableWidth := width - 6 // Padding (2) + Border (2) + Safe gap (2)

		// Render on one line
		var content string
		if a.Progress < 0 {
			// Indeterminate mode: Title + Warning aligned right

			// Bird-nerdy warning per tone_of_voice.md
			warningText := "Steady now... keep the nest open."
			warningPart := t.Styles.TextMuted.Italic(true).Render(warningText)

			// We need to calculate how much space the title CAN take vs how much generic gap we have.
			// Ideally we just flex space between them.

			// Combine spinner + msg for width check
			// We use a layout style just for width, no color (so internal colors persist)
			titleText := fmt.Sprintf("%s %s", spinnerView, messageView)

			gapWidth := availableWidth - lipgloss.Width(titleText) - lipgloss.Width(warningPart)
			if gapWidth < 1 {
				gapWidth = 1
			}

			content = lipgloss.JoinHorizontal(lipgloss.Center,
				titleText,
				strings.Repeat(" ", gapWidth),
				warningPart,
			)
		} else {
			// Standard mode: Title + Bar
			// Calculate widths for single-line
			// [Spinner] MESSAGE ................. [BAR]

			// Allocate 40% to title block, 60% to bar
			titleWidth := availableWidth * 4 / 10
			if titleWidth < 20 {
				titleWidth = 20
			} // Min width for title

			barWidth := availableWidth - titleWidth - 2 // space
			if barWidth < 10 {
				barWidth = 10
			}

			a.Bar.Width = barWidth
			progressBar := a.Bar.ViewAs(val)

			// We wrap the Title Text in a container just to enforce Width.
			// We do NOT apply foreground color here, to respect the internal styles.
			titleText := fmt.Sprintf("%s %s", spinnerView, messageView)
			titlePart := lipgloss.NewStyle().Width(titleWidth).Render(titleText)

			content = lipgloss.JoinHorizontal(lipgloss.Center,
				titlePart,
				"  ", // Gap
				progressBar,
			)
		}

		rows = append(rows, cardStyle.Render(content))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
