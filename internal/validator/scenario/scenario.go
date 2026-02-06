package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/validator/config"
	"github.com/Josepavese/nido/internal/validator/report"
	"github.com/Josepavese/nido/internal/validator/runner"
	"github.com/Josepavese/nido/internal/validator/state"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Context is shared across scenarios.
type Context struct {
	RunID    string
	Config   config.Config
	Runner   runner.Runner
	State    *state.State
	Reporter *report.Reporter
	Vars     map[string]string
	Failures []report.StepResult
	Start    time.Time
	Total    int
	Index    int
}

// Step is a single execution unit.
type Step func(*Context) report.StepResult

// Scenario bundles related steps.
type Scenario struct {
	Name  string
	Steps []Step
}

// Run executes scenarios sequentially and writes results via reporter.
func Run(ctx *Context, scenarios []Scenario) {
	for _, sc := range scenarios {
		printScenarioHeader(sc.Name)
		for idx, step := range sc.Steps {
			if ctx.Config.FailFast && ctx.Reporter.Summary.Fail > 0 && sc.Name != "cleanup" {
				return
			}
			ctx.Index++
			stepID := sc.Name + "-" + itoa(idx+1)
			res := step(ctx)
			res.RunID = ctx.RunID
			res.StepID = stepID
			res.Scenario = sc.Name
			_ = ctx.Reporter.WriteStep(res)
			if res.Result == "FAIL" {
				ctx.Failures = append(ctx.Failures, res)
			}
			printStepLine(ctx, sc.Name, res)
		}
		printScenarioFooter()
	}
}

func itoa(v int) string {
	return fmt.Sprintf("%d", v)
}

var (
	renderer      = lipgloss.NewRenderer(os.Stdout)
	width         = 100
	hasOutput     = false
	titleStyle    = renderer.NewStyle().Foreground(lipgloss.Color("201")).Bold(true)
	progressStyle = renderer.NewStyle().Foreground(lipgloss.Color("51")).Bold(true)
	sectionStyle  = renderer.NewStyle().Foreground(lipgloss.Color("45")).Bold(true)
	passStyle     = renderer.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	failStyle     = renderer.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	skipStyle     = renderer.NewStyle().Foreground(lipgloss.Color("227")).Bold(true)
	cmdStyle      = renderer.NewStyle().Foreground(lipgloss.Color("252"))
	msStyle       = renderer.NewStyle().Foreground(lipgloss.Color("245"))
)

func printScenarioHeader(name string) {
	if !hasOutput {
		if w := detectWidth(); w > 0 {
			width = w
		}
	}
	if hasOutput {
		fmt.Println()
	}
	hasOutput = true
	title := fmt.Sprintf(" SCENARIO: %s ", strings.ToUpper(name))
	inner := width - 2
	fill := max(0, inner-len(stripANSI(title)))
	fmt.Printf("┌%s%s┐\n", titleStyle.Render(title), strings.Repeat("─", fill))
}

func printScenarioFooter() {
	fmt.Printf("└%s┘\n", strings.Repeat("─", max(0, width-2)))
}

func printStepLine(ctx *Context, scenarioName string, res report.StepResult) {
	progress := fmt.Sprintf("[%02d/%02d]", ctx.Index, ctx.Total)
	badge, badgeStyle, icon := "PASS", passStyle, "✔"
	if res.Result == "FAIL" {
		badge, badgeStyle, icon = "GLITCH", failStyle, "✖"
	} else if res.Result == "SKIP" {
		badge, badgeStyle, icon = "SKIP", skipStyle, "…"
	}
	cmd := filepath.Base(res.Command)
	detail := ""
	if len(res.Args) > 0 {
		detail = strings.Join(res.Args, " ")
	}
	inner := width - 2
	base := 1 + runeLen(icon) + 1 + runeLen(progress) + 1 + runeLen(strings.ToUpper(scenarioName)) + 1 + runeLen(badge) + 1 + runeLen(fmt.Sprintf("%d ms", res.DurationMs)) + 1
	cmdBudget := inner - base
	if cmdBudget < 0 {
		cmdBudget = 0
	}
	cmdText := truncateRunes(strings.TrimSpace(fmt.Sprintf("%s %s", cmd, detail)), cmdBudget)

	pad := inner - runeLen(fmt.Sprintf(" %s %s %s %s %s %d ms", icon, progress, strings.ToUpper(scenarioName), badge, cmdText, res.DurationMs))
	if pad < 0 {
		pad = 0
	}

	styled := lipgloss.JoinHorizontal(lipgloss.Left,
		" ",
		badgeStyle.Render(icon),
		" ",
		progressStyle.Render(progress),
		" ",
		sectionStyle.Render(strings.ToUpper(scenarioName)),
		" ",
		badgeStyle.Render(badge),
		" ",
		cmdStyle.Render(cmdText),
		strings.Repeat(" ", pad),
		msStyle.Render(fmt.Sprintf("%d ms", res.DurationMs)),
	)
	fmt.Printf("│%s│\n", styled)
}

func detectWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w >= 60 && w <= 200 {
		return w
	}
	if v := os.Getenv("COLUMNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 60 && n <= 200 {
			return n
		}
	}
	return 0
}

func runeLen(s string) int {
	return len([]rune(stripANSI(s)))
}

func truncateRunes(s string, maxRunes int) string {
	plain := []rune(stripANSI(s))
	if maxRunes <= 0 {
		return ""
	}
	if len(plain) <= maxRunes {
		return string(plain)
	}
	if maxRunes <= 3 {
		return string(plain[:maxRunes])
	}
	return string(plain[:maxRunes-3]) + "..."
}

func stripANSI(s string) string {
	out := make([]rune, 0, len(s))
	skip := false
	for _, r := range s {
		if r == 0x1b {
			skip = true
			continue
		}
		if skip {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				skip = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
