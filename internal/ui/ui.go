package ui

import (
	"fmt"
	"strings"
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
	IconBird    = "🐣"
	IconEgg     = "🥚"
	IconRocket  = "🚀"
	IconStop    = "⏹️"
	IconTrash   = "🗑️"
	IconInfo    = "📋"
	IconConfig  = "⚙️"
	IconSuccess = "✅"
	IconWarning = "⚠️"
	IconError   = "❌"
	IconPulse   = "⚡"
)

func Header(title string) {
	fmt.Printf("\n%s%s%s %s%s%s\n", Bold, Blue, IconBird, strings.ToUpper(title), Reset, Reset)
	fmt.Printf("%s%s%s\n\n", Dim, strings.Repeat("-", 40), Reset)
}

func Info(msg string, args ...interface{}) {
	fmt.Printf("%s%s%s %s\n", Cyan, IconInfo, Reset, fmt.Sprintf(msg, args...))
}

func Success(msg string, args ...interface{}) {
	fmt.Printf("%s%s%s %s\n", Green, IconSuccess, Reset, fmt.Sprintf(msg, args...))
}

func Warn(msg string, args ...interface{}) {
	fmt.Printf("%s%s%s %s\n", Yellow, IconWarning, Reset, fmt.Sprintf(msg, args...))
}

func Error(msg string, args ...interface{}) {
	fmt.Printf("%s%s%s %s%s%s\n", Red, IconError, Bold, fmt.Sprintf(msg, args...), Reset, Reset)
}

func FancyLabel(label string, value interface{}) {
	fmt.Printf("  %s%-15s%s %v\n", Cyan, label, Reset, value)
}

func Ironic(msg string) {
	fmt.Printf("%s%s %s%s\n", Dim, IconPulse, msg, Reset)
}

func DoctorCheck(label string, passed bool, details string) {
	status := Green + " [PASS] " + Reset
	icon := IconSuccess
	if !passed {
		status = Red + Bold + " [FAIL] " + Reset
		icon = IconError
	}
	fmt.Printf("  %s %-20s %s %s%s%s\n", icon, label, status, Dim, details, Reset)
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
