//go:build windows

package provider

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const windowsCommandLineLimit = 128 * 1024

func inspectHandle(handle windows.Handle, identity vmProcessIdentity) error {
	var size uint32
	err := windows.NtQueryInformationProcess(handle, windows.ProcessCommandLineInformation, nil, 0, &size)
	if size == 0 || size > windowsCommandLineLimit {
		return fmt.Errorf("query VM command line size: %v", err)
	}
	buffer := make([]byte, size)
	if err := windows.NtQueryInformationProcess(handle, windows.ProcessCommandLineInformation, unsafe.Pointer(&buffer[0]), size, &size); err != nil {
		return err
	}
	if len(buffer) < int(unsafe.Sizeof(windows.NTUnicodeString{})) {
		return fmt.Errorf("invalid process command line")
	}
	value := (*windows.NTUnicodeString)(unsafe.Pointer(&buffer[0]))
	args, err := windows.DecomposeCommandLine(windows.UTF16ToString(unsafe.Slice(value.Buffer, int(value.Length)/2)))
	runtime.KeepAlive(buffer)
	if err != nil {
		return err
	}
	exe := make([]uint16, windows.MAX_PATH*4)
	length := uint32(len(exe))
	if err := windows.QueryFullProcessImageName(handle, 0, &exe[0], &length); err != nil {
		return err
	}
	if !qemuExecutable(windows.UTF16ToString(exe[:length])) || !matchesVMCommand(args, identity) {
		return fmt.Errorf("process does not belong to VM %q", identity.name)
	}
	return nil
}

func inspectVMProcess(pid int, identity vmProcessIdentity) error {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	return inspectHandle(handle, identity)
}

func stopManagedProcess(pid int, identity vmProcessIdentity, _ bool) error {
	// Inspect and terminate the same kernel handle, never a reopened numeric PID.
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	if err := inspectHandle(handle, identity); err != nil {
		return err
	}
	if err := windows.TerminateProcess(handle, 0); err != nil {
		return err
	}
	status, err := windows.WaitForSingleObject(handle, uint32(vmStopTimeout.Milliseconds()))
	if err != nil {
		return err
	}
	if status != windows.WAIT_OBJECT_0 {
		return fmt.Errorf("VM %q did not exit", identity.name)
	}
	return nil
}
