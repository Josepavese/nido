package ops

import (
	"context"
	"time"

	"github.com/Josepavese/nido/v4/internal/events"
	tea "github.com/charmbracelet/bubbletea"
)

// EventBatchMsg carries one coalesced invalidation; blocked event receives end
// with the owning TUI context. No domain refresh timer runs in the TUI.
type EventBatchMsg struct{ Events []events.Event }

func WaitEvents(ctx context.Context, stream <-chan events.Event) tea.Cmd {
	if stream == nil {
		return nil
	}
	return func() tea.Msg {
		var first events.Event
		select {
		case <-ctx.Done():
			return nil
		case e, ok := <-stream:
			if !ok {
				return nil
			}
			first = e
		}
		batch := []events.Event{first}
		timer := time.NewTimer(events.CoalesceWindow)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				return EventBatchMsg{batch}
			case e, ok := <-stream:
				if !ok {
					return EventBatchMsg{batch}
				}
				if len(batch) < events.QueueSize {
					batch = append(batch, e)
				} else {
					batch = []events.Event{{Source: events.Service, Kind: events.Resync}}
				}
			}
		}
	}
}
