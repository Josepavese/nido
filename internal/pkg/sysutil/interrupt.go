package sysutil

import (
	"context"
	"os"
	"os/signal"
)

// InterruptContext owns a portable interactive command cancellation registration.
func InterruptContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt)
}
