//go:build !linux

package hardware

import "context"

func (*nativeHost) ListAccelerators() ([]Accelerator, error) { return nil, ErrUnsupported }
func (*nativeHost) PreparePassthrough(context.Context, string, string, string) error {
	return ErrUnsupported
}
func (*nativeHost) RestorePassthrough(context.Context, string, string, string) error {
	return ErrUnsupported
}
