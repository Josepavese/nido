package sysutil

import (
	"context"
	"os"
	"strconv"
)

// AllowMemoryLock adjusts only this Nido process's inherited limit before it
// launches a VFIO VM; commands and elevation still use the injected PAL.
func AllowMemoryLock(ctx context.Context, io HostIO) error {
	return RunPrivileged(ctx, io, "prlimit", "--pid", strconv.Itoa(os.Getpid()), "--memlock=unlimited:unlimited")
}
