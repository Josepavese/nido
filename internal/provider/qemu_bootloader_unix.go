//go:build !windows

package provider

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"time"
)

const qemuBootMenuDelay = 3 * time.Second
const qemuBootKeyDelay = time.Second
const qemuBootDialTimeout = 500 * time.Millisecond
const qemuBootControlTimeout = 5 * time.Second
const qemuBootKeyAttempts = 3

// skipBootloader is our secret weapon for speed. It connects via QMP
// and mashes the "Enter" key while the VM is starting up to bypass
// guest bootloader menus.
func (p *QemuProvider) skipBootloader(name string) {
	qmpPath := qemuQMPPath(filepath.Join(p.RootDir, "run"), name)
	if qmpPath == "" {
		return
	}

	// 1. Initial wait: Give BIOS/UEFI time to finish and reach bootloader (3s)
	time.Sleep(qemuBootMenuDelay)

	// 2. Connect to QMP
	conn, err := net.DialTimeout("unix", qmpPath, qemuBootDialTimeout)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(qemuBootControlTimeout))

	// 3. Handshake
	decoder := json.NewDecoder(conn)
	var greeting map[string]interface{}
	_ = decoder.Decode(&greeting)
	fmt.Fprintf(conn, `{"execute":"qmp_capabilities"}`+"\n")
	_ = decoder.Decode(&greeting)

	// 4. Send "Return" exactly 3 times with 1s gap
	// This covers potential UI lag or early bootloader states
	for i := 0; i < qemuBootKeyAttempts; i++ {
		cmd, _ := json.Marshal(map[string]interface{}{
			"execute": "send-key",
			"arguments": map[string]interface{}{
				"keys": []map[string]interface{}{
					{"type": "qcode", "data": "ret"},
				},
			},
		})
		fmt.Fprintf(conn, "%s\n", string(cmd))

		var res map[string]interface{}
		_ = decoder.Decode(&res)

		time.Sleep(qemuBootKeyDelay)
	}
}

// ListImages scans the image directory for cached images.
