package nidonet

import (
	"fmt"
	"net"
	"time"
)

// FindAvailablePort scans the given range for a free TCP port, skipping reserved ones.
// Implements Section 5.4.B of advanced-port-forwarding.md.
func FindAvailablePort(start, end int, reserved map[int]bool) (int, error) {
	for port := start; port <= end; port++ {
		if reserved != nil && reserved[port] {
			continue
		}
		if IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}

// IsPortAvailable checks if a TCP port is available on localhost.
func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()

	// Brief pause to ensure OS releases it fully, though Close() is usually enough
	time.Sleep(1 * time.Millisecond)
	return true
}
