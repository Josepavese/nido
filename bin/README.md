# VM Ops Scripts

Executable entrypoints for VM lifecycle operations.

Main script:
- `nido` : spawn/create/start/stop/delete/ls/info/prune/template/selftest

Template creation:
- handled by `nido template`

Install prerequisites:
- `setup_deps.sh` : installs dependencies (libvirt/qemu/virt-install) and verifies environment (Linux/macOS).
