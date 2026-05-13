# Windows Support

Nido now has two Windows-facing surfaces:

- Windows as a host OS for running Nido itself.
- Windows images as buildable blueprints.

## Host Status

Windows host support is smoke-tested on a real Windows VM. The latest validation covered:

- Windows PowerShell 5.1 parsing of `installers/quick-install.ps1`.
- `nido doctor`.
- image and blueprint catalog listing.
- basic VM lifecycle from Windows: spawn, SSH command execution, stop, delete.

This is enough to call the core workflow usable. It is not yet equivalent to the Linux/macOS path for heavy or long-running workloads. Keep testing intensive agent workloads, repeated lifecycle loops, GUI sessions, and large Windows blueprint builds before relying on it as the default production host.

## Installer Behavior

`quick-install.ps1` installs the Nido release into `%USERPROFILE%\.nido`, adds `%USERPROFILE%\.nido\bin` to the user PATH, and installs the bundled registry.

For VM prerequisites it:

- checks for `qemu-system` and `qemu-img`;
- checks for `winget`;
- registers Microsoft App Installer when present but not registered;
- downloads and installs Microsoft App Installer when winget is missing;
- installs QEMU through winget when the user confirms;
- asks whether to enable Windows Hypervisor Platform for WHPX acceleration.

WHPX enablement requires administrator approval and a Windows restart. If WHPX is unavailable, Nido can fall back to QEMU TCG.

## Windows Blueprints

Available entries:

| Blueprint | Product |
| --- | --- |
| `windows-11-eval` | Windows 11 Enterprise Evaluation |
| `windows-11-iot-ltsc-eval` | Windows 11 IoT Enterprise LTSC 2024 Evaluation |
| `windows-server-2022-core-eval` | Windows Server 2022 Evaluation (Server Core) |

Example:

```bash
nido blueprint list
nido blueprint build windows-11-eval
nido spawn win11 --image windows-11-eval --gui
```

The blueprint build prepares the image. The first boot can still run Windows OOBE/post-install work, so use `--gui` for the first start.
