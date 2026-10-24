package sysutil

import (
	"errors"
	"time"
)

// Native process waits block in the kernel. This bound only limits cancellation
// latency on backends whose wait cannot receive a Go context directly.
const processWaitCancelInterval = 100 * time.Millisecond

var errInvalidProcessIdentity = errors.New("process identity must be a positive PID")
