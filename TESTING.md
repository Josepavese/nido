# Beta Testing Guide for Nido v3

Thank you for helping test Nido's cross-platform support! This guide will help you test Nido on macOS and Windows.

## Prerequisites

### macOS

- macOS 10.13 (High Sierra) or later
- Homebrew installed
- At least 4GB free RAM

### Windows

- Windows 10/11 (64-bit)
- Administrator privileges
- At least 4GB free RAM

## Installation

### macOS

```bash
# Install QEMU via Homebrew
brew install qemu

# Download Nido binary (replace with actual release URL)
curl -L https://github.com/Josepavese/nido/releases/download/v3.0.0/nido-darwin-amd64 -o nido
chmod +x nido
sudo mv nido /usr/local/bin/

# Verify installation
nido version
nido doctor
```

### Windows

```powershell
# Install QEMU via Chocolatey
choco install qemu

# Download Nido binary (replace with actual release URL)
# Download from GitHub releases page
# Move to C:\Program Files\nido\nido.exe

# Add to PATH
$env:Path += ";C:\Program Files\nido"

# Verify installation
nido version
nido doctor
```

## Testing Checklist

Please test the following commands and report any issues:

### ✅ Basic Commands (All Platforms)

- [ ] `nido version` - Shows version info
- [ ] `nido doctor` - System diagnostics
- [ ] `nido config` - Shows configuration
- [ ] `nido template list` - Lists templates (may be empty)
- [ ] `nido ls` - Lists VMs (may be empty)

### ✅ VM Lifecycle (If you have a template)

- [ ] `nido spawn test-vm` - Create and start a VM
- [ ] `nido ls` - Verify VM appears
- [ ] `nido info test-vm` - Get VM details
- [ ] `nido ssh test-vm` - SSH connection string
- [ ] `nido stop test-vm` - Stop the VM
- [ ] `nido start test-vm` - Restart the VM
- [ ] `nido delete test-vm` - Remove the VM

### ✅ MCP Server (Advanced)

- [ ] `nido register` - Shows MCP configuration
- [ ] Test MCP server with Claude/Antigravity

## Known Limitations

### macOS

- **Hypervisor Framework (HVF)** requires macOS 10.13+
- First VM start may be slow (QEMU initialization)
- No GUI tools like virt-manager

### Windows

- **WHPX** requires Windows 10 build 19041+ with Hyper-V enabled
- May conflict with other hypervisors (VirtualBox, VMware)
- QMP uses TCP instead of Unix sockets

### All Platforms

- VMs use **localhost port forwarding** instead of direct IP
- SSH via `ssh -p <port> user@localhost` (port shown in `nido info`)
- No nested virtualization support on GitHub Actions runners

## Reporting Issues

Please report issues on GitHub with:

1. **Platform**: macOS/Windows version
2. **QEMU version**: Output of `qemu-system-x86_64 --version`
3. **Command**: Exact command that failed
4. **Error**: Full error message
5. **Logs**: Output of `nido doctor`

Example issue:

```
Platform: macOS 14.2 (Sonoma)
QEMU: 8.1.0
Command: nido spawn test-vm
Error: "qemu-system-x86_64: -accel hvf: invalid accelerator hvf"
Doctor output: [paste here]
```

## Success Criteria

For a successful beta test, we need:

- ✅ `nido doctor` passes on your platform
- ✅ At least one successful VM spawn/start/stop cycle
- ✅ SSH connection works (even if you can't login)

## Getting Help

- GitHub Issues: <https://github.com/Josepavese/nido/issues>
- Discussions: <https://github.com/Josepavese/nido/discussions>

Thank you for testing! 🚀
