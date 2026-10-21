package sysutil

import "errors"

func (NativeHostIO) DeviceValue(string, uintptr) (uintptr, error) { return 0, errors.ErrUnsupported }
