# Windows blueprints fail before downstream installer testing

Date: 2026-06-02

## Summary

The available Windows blueprints do not currently produce a bootable Windows VM suitable for downstream installer tests.

This was found while trying to validate a published Needlex Windows installer through Nido. The Needle-X release was already validated on Linux; the Windows validation was blocked before Needlex could be installed because Nido could not build a usable Windows image from any available Windows blueprint.

This issue documents the Nido-side failure only. No Nido code changes were made.

## Environment

- Host: Linux x86_64
- Nido path: `/home/jose/.nido/bin/nido`
- QEMU/KVM: available
- `nido doctor`: passed for directories, QEMU, qemu-img, ISO creator, and KVM accessibility
- Existing Windows blueprints:
  - `windows-11-eval`
  - `windows-11-iot-ltsc-eval`
  - `windows-server-2022-core-eval`

## Impact

Nido cannot currently provide a Windows VM for automated install validation.

This blocks downstream projects from testing Windows installers with Nido, even when the project installer itself may be correct.

Expected behavior:

- `nido build <windows-blueprint>` should produce a bootable image, or fail with an actionable Nido error and logs.
- The resulting VM should be reachable over SSH as declared by the blueprint.

Actual behavior:

- All available Windows blueprints failed before a usable VM image was produced.

## Reproduction

```bash
/home/jose/.nido/bin/nido doctor
/home/jose/.nido/bin/nido blueprint list
/home/jose/.nido/bin/nido build windows-11-eval --json
/home/jose/.nido/bin/nido build windows-11-iot-ltsc-eval --json
/home/jose/.nido/bin/nido build windows-server-2022-core-eval --json
```

During QEMU runs, the build VM exposed VNC on:

```text
127.0.0.1:5999
```

Screenshots were captured manually from the VNC framebuffer for diagnosis.

## Observed failures

### `windows-11-eval`

QEMU started from the cached ISO:

```text
/home/jose/.nido/cache/windows-11-enterprise-eval.iso
```

The installer eventually stopped at a Windows setup error dialog:

```text
Windows 11 installation has failed
```

No usable final image was produced.

### `windows-11-iot-ltsc-eval`

The ISO download completed and QEMU started from:

```text
/home/jose/.nido/cache/windows-11-iot-ltsc-eval.iso
```

The seed ISO existed:

```text
/home/jose/.nido/tmp/windows-11-iot-ltsc-eval-seed.iso
```

The seed ISO contained:

```text
/autounattend.xml
```

The answer file variables were substituted correctly, including:

```xml
<Value>1</Value>
```

and the `w11` VirtIO profile paths.

However, Windows setup did not appear to consume the answer file. The VNC framebuffer showed the interactive Windows 11 language selection screen in Chinese, and the target qcow2 stayed at approximately 324 KB:

```text
/home/jose/.nido/images/windows-11-iot-ltsc.qcow2.<pid>.building
```

No disk install started and no usable final image was produced.

### `windows-server-2022-core-eval`

The ISO download completed:

```text
/home/jose/.nido/cache/windows-server-2022-core-eval.iso
```

QEMU started and loaded Windows setup, but setup stopped with:

```text
Windows cannot find the Microsoft Software License Terms.
Make sure the installation sources are valid and restart the installation.
```

The ISO contains:

```text
sources/install.wim
sources/EI.CFG
```

`EI.CFG` contents:

```ini
[Channel]
eval

[VL]
0
```

No usable final image was produced.

## Additional notes

The Windows 11 IoT failure is notable because the generated seed ISO does contain `/autounattend.xml`, and variable substitution appears correct. The most likely area to inspect is how Windows setup discovers the answer file from the attached seed ISO.

Possible Nido-side investigation points:

- CD-ROM drive ordering and whether Windows setup scans the seed ISO for `Autounattend.xml`.
- Case sensitivity or filename expectations: `autounattend.xml` vs `Autounattend.xml`.
- ISO filesystem options generated for the seed ISO.
- Whether the selected Microsoft ISO redirects to a localized installer that changes answer-file behavior.
- Whether Windows 11 setup requires different boot or answer-file placement for this media.

For `windows-server-2022-core-eval`, likely investigation points:

- `windows_image_index` may not match the intended Server Core evaluation image.
- The generated unattended file may need edition metadata beyond `/IMAGE/INDEX`.
- The evaluation ISO license files or edition selection may not be compatible with the current unattended block.

## Cleanup performed

Only incomplete build artifacts generated during this investigation were removed:

```text
/home/jose/.nido/images/*.building
```

No Nido source files were modified and no commit was made.
