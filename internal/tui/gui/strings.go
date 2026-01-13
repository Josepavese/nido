package gui

// UIStrings centralizes user-facing labels for easier override/testing.
type UIStrings struct {
	TabLabels  []string
	FooterLink string
}

// DefaultStrings returns the default UI strings.
func DefaultStrings() UIStrings {
	return UIStrings{
		TabLabels:  []string{"1 FLEET", "2 HATCHERY", "3 LOGS", "4 CONFIG", "5 HELP"},
		FooterLink: "https://github.com/Josepavese",
	}
}
