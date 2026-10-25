package widget

import (
	"fmt"
	"github.com/Josepavese/nido/v4/internal/tui/kit/layout"
	"github.com/Josepavese/nido/v4/internal/tui/kit/theme"
	"strings"
)

// ScrollView owns a vertical viewport. An optional anchor follows keyboard focus;
// manual scrolling remains in place until focus or geometry actually changes.
type ScrollView struct {
	offset, width, height, contentHeight int
	anchor                               layout.Rect
}

func (s *ScrollView) Scroll(lines int) {
	s.offset = max(0, min(s.offset+lines, s.contentHeight-s.height))
}
func (s *ScrollView) Render(width, height int, content string, anchor layout.Rect) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	visible := height
	overflow := len(lines) > height
	if overflow {
		visible = max(1, height-layout.ScrollHintHeight)
	}
	if anchor != s.anchor || width != s.width || visible != s.height {
		if anchor.Height > 0 {
			if anchor.Y < s.offset {
				s.offset = anchor.Y
			}
			if anchor.Y+anchor.Height > s.offset+visible {
				s.offset = anchor.Y + min(anchor.Height, visible) - visible
				if anchor.Height > visible {
					s.offset = anchor.Y + (anchor.Height-visible)/2
				}
			}
		}
	}
	s.width, s.height, s.contentHeight, s.anchor = width, visible, len(lines), anchor
	s.offset = max(0, min(s.offset, len(lines)-visible))
	end := min(len(lines), s.offset+visible)
	result := strings.Join(lines[s.offset:end], "\n")
	if overflow && height > layout.ScrollHintHeight {
		indicator := fmt.Sprintf("↑↓ %d-%d/%d · pgup/pgdn", s.offset+1, end, len(lines))
		result = layout.Fit(width, visible, result) + "\n" + theme.Current().Styles.TextMuted.Render(indicator)
	}
	return layout.Fit(width, height, result)
}
