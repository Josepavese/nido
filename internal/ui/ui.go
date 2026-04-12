package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	clijson "github.com/Josepavese/nido/internal/cli"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
	Purple  = "\033[35m" // Alias for Magenta
)

const (
	IconBird    = "N"
	IconEgg     = "o"
	IconRocket  = ">"
	IconStop    = "x"
	IconTrash   = "-"
	IconInfo    = "i"
	IconConfig  = "="
	IconSuccess = "+"
	IconWarning = "!"
	IconError   = "x"
	IconPulse   = ">"
)

func Header(title string) {
	if silent() {
		return
	}
	writef(os.Stdout, "\n%sNIDO%s  %s%s%s\n", Bold+Blue, Reset, Bold, strings.ToUpper(title), Reset)
	writef(os.Stdout, "%s%s%s\n\n", Dim, strings.Repeat("─", 56), Reset)
}

func Info(msg string, args ...interface{}) {
	line("info", Cyan, msg, args...)
}

func Success(msg string, args ...interface{}) {
	line("done", Green, msg, args...)
}

func Warn(msg string, args ...interface{}) {
	line("warn", Yellow, msg, args...)
}

func Error(msg string, args ...interface{}) {
	line("fail", Red, msg, args...)
}

func FancyLabel(label string, value interface{}) {
	if silent() {
		return
	}
	writef(os.Stdout, "  %s%-16s%s %v\n", Cyan, label, Reset, value)
}

func Ironic(msg string) {
	Step("%s", msg)
}

func Step(msg string, args ...interface{}) {
	line("step", Dim, msg, args...)
}

func Section(title string) {
	if silent() {
		return
	}
	writef(os.Stdout, "%s%s%s\n", Bold, title, Reset)
}

func TableHeader(columns ...string) {
	if silent() {
		return
	}
	writef(os.Stdout, "\n %s%s%s\n", Bold, strings.Join(columns, "  "), Reset)
}

func Rule(width int) {
	if silent() {
		return
	}
	if width <= 0 {
		width = 56
	}
	writef(os.Stdout, " %s%s%s\n", Dim, strings.Repeat("─", width), Reset)
}

func DoctorCheck(label string, passed bool, details string) {
	if silent() {
		return
	}
	status := Green + " [PASS] " + Reset
	icon := IconSuccess
	if !passed {
		status = Red + Bold + " [FAIL] " + Reset
		icon = IconError
	}
	writef(os.Stdout, "  %s %-20s %s %s%s%s\n", icon, label, status, Dim, details, Reset)
}

func HumanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func line(label, color, msg string, args ...interface{}) {
	if silent() {
		return
	}
	writef(os.Stdout, "%s%-4s%s %s\n", color, label, Reset, fmt.Sprintf(msg, args...))
}

func silent() bool {
	return clijson.IsJSONMode()
}

func writef(w io.Writer, format string, args ...interface{}) {
	_, _ = fmt.Fprintf(w, format, args...)
}
