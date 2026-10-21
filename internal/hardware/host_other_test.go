//go:build !linux

package hardware

import (
	"context"
	"errors"
	"testing"
)

func TestUnsupportedPCICapabilityIsExplicit(t *testing.T) {
	h := New()
	if _, err := h.ListAccelerators(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("inventory: %v", err)
	}
	if err := h.PreparePassthrough(context.Background(), "0000:01:00.0", "unused", "fixture"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("prepare: %v", err)
	}
	if err := h.RestorePassthrough(context.Background(), "0000:01:00.0", "unused", "fixture"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("restore: %v", err)
	}
}
