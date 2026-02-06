package fleet

import (
	"fmt"

	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/tui/kit/theme"
)

// AccelItem adapts provider.Accelerator to the List interface.
// Duplicated from hatchery to avoid import cycles for now.
type AccelItem struct {
	Acc provider.Accelerator
}

func (i AccelItem) Title() string {
	return fmt.Sprintf("[%s] %s", i.Acc.ID, i.Acc.Class)
}
func (i AccelItem) Description() string {
	status := "Safe"
	if !i.Acc.IsSafe {
		status = "UNSAFE: " + i.Acc.Warning
	}
	return fmt.Sprintf("%s | Grp: %s", status, i.Acc.IOMMUGroup)
}
func (i AccelItem) FilterValue() string { return i.Acc.ID + " " + i.Acc.Class }
func (i AccelItem) String() string      { return i.Title() }
func (i AccelItem) Icon() string        { return theme.IconHatchery } // Generic chip icon?
func (i AccelItem) IsAction() bool      { return false }
