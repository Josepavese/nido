package gui

// Keymap centralizes keybindings for tabs and global actions.
type Keymap struct {
	TabNext   string
	TabPrev   string
	Quit      string
	Refresh   string
	TabSelect []string // e.g., []string{"1","2","3","4","5"}
}

// DefaultKeymap returns the default keyboard shortcuts.
func DefaultKeymap() Keymap {
	return Keymap{
		TabNext:   "right",
		TabPrev:   "left",
		Quit:      "q",
		Refresh:   "r",
		TabSelect: []string{"1", "2", "3", "4", "5"},
	}
}
