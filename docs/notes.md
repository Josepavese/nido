# VM Ops Notes

- 2025-01-03: `virsh` access failed in current shell with "Permission denied" on `/var/run/libvirt/libvirt-sock` even though user is in `libvirt`/`kvm`. `sg libvirt -c "virsh list --all"` worked; logout/login likely required.
- 2025-01-03: `debian-headless-1` had no IP via `virsh domifaddr` and ping to `192.168.56.10` failed. Issue traced to guest network config.
- 2025-01-04: Headless template updated with `qemu-guest-agent` for reliable hostname/IP detection.
- 2025-01-04: VM lifecycle consolidated into `nido` with volume deletion via `virsh vol-delete`.
- 2025-01-04: Command names normalized (spawn/create/ls/delete/prune/template) and selftest verified via mock.
- 2025-01-04: Templates are stored as compressed backups (`.compact.qcow2`) under `BACKUP_DIR`.
- 2025-01-04: Centralized configuration via `./config/config.env` (see `config.example.env`).
