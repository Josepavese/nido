# VM Ops (libvirt + KVM)

This folder provides a minimal, automation-friendly toolkit to manage headless
VMs for DaemonZero workloads. The workflow is built around **compressed
template backups** (cold storage) so you can keep disk usage low while still
deploying new VMs quickly.

## Folder layout
- `./bin/` : runnable scripts
- `./config/` : configuration templates
- `./docs/` : notes and support docs

## Philosophy: compressed templates only
Templates are stored as `.compact.qcow2` in the backups directory and **no live
template disks** are kept. New VMs are created by expanding a compressed
template into a fresh qcow2 disk.

## Quick start
1) Configure:
   - `cp ./config/config.example.env ./config/config.env`
   - edit `./config/config.env`
2) Install prerequisites (one-time):
   - `./bin/install.sh`
3) Create a compressed template from a base VM (VM must be shut off):
   - `./bin/nido template debian-iso-1 template-headless`
4) Create a VM:
   - `./bin/nido create vm-test-1`
5) Start and wait for IP:
   - `./bin/nido start vm-test-1`

## Configuration
Default config file:
- `./config/config.env`

You can also override with:
- `VMOPS_CONFIG=/path/to/config.env`

Key variables:
- `POOL_PATH=/media/jose/Data/libvirt-pool`
- `BACKUP_DIR=/media/jose/Data/libvirt-pool/backups`
- `VMS_POOL=vms`
- `TEMPLATE_DEFAULT=template-headless`
- `VM_MEM_MB=2048`
- `VM_VCPUS=2`
- `VM_OS_VARIANT=debian12`
- `NETWORK_HOSTONLY=hostonly56`
- `NETWORK_NAT=default`
- `SSH_USER=vmuser`
- `WAIT_TIMEOUT=60`
- `GRAPHICS=spice` (spice|vnc|none)

## Commands (nido)

All commands should be run via `./bin/nido <command>`.

### VM Lifecycle

**Spawn (Primary)**
Create and start a VM from a compressed template backup in one command:
```bash
./bin/nido spawn vm-test-1
```

**Create**
Just create the VM disk and define the libvirt domain without starting it:
```bash
./bin/nido create vm-test-1
```

**Start**
Start an existing VM (or create it if it doesn't exist):
```bash
./bin/nido start vm-test-1
```

**Stop**
Gracefully shutdown a running VM:
```bash
./bin/nido stop vm-test-1
```

**Delete**
Permanently remove a VM and its associated disk volume:
```bash
./bin/nido delete vm-test-1
```

### Observation & Management

**List (ls)**
List all VMs (matches all by default, or use a regex to filter):
```bash
./bin/nido ls
./bin/nido ls '^web-'
```

**Info**
Print detailed connection info (IP, Hostname, SSH command):
```bash
./bin/nido info vm-test-1
```

**Prune**
Identify and delete orphaned disk volumes in the `vms` pool:
```bash
./bin/nido prune
```

### Template Management

**Template**
Create a compressed backup (`.compact.qcow2`) from an existing VM (must be stopped):
```bash
./bin/nido template my-vm template-name
```

### Diagnostics

**Selftest**
Run automated tests to verify the environment.
```bash
./bin/nido selftest         # Real tests (checks libvirt)
./bin/nido selftest --mock  # Mocked tests (no side effects)
```
