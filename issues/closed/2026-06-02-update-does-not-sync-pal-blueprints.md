# Nido update does not sync bundled blueprints into PAL home

Date: 2026-06-02

## Summary

`nido update` updated the Nido binary from `v4.5.22` to `v4.5.23`, but it did not update the bundled Windows blueprint files already present under the PAL home registry:

```text
/home/jose/.nido/registry/blueprints
```

This left the runtime using stale blueprint definitions even after the Nido binary containing the Windows provisioning fix had been installed.

No Nido code changes were made as part of this note. This file only documents the observed behavior for the team that will implement the correction.

## Environment

- Host: Linux x86_64
- PAL home: `/home/jose/.nido`
- Nido binary: `/home/jose/.nido/bin/nido`
- Previous version: `v4.5.22`
- Updated version: `v4.5.23`
- Relevant local source checkout: `/home/jose/hpdev/Libraries/nido`

## Trigger

The issue was found while retrying Windows VM creation after the Nido-side Windows blueprint fix.

The command:

```bash
/home/jose/.nido/bin/nido update
```

reported:

```text
Found new version: v4.5.23 (current: v4.5.22)
Updated to v4.5.23.
Shell completions updated in /home/jose/.nido.
```

After that update, the PAL-home blueprint files were still the old definitions.

## Observed Behavior

After the binary update, these files remained stale:

```text
/home/jose/.nido/registry/blueprints/windows-11-iot-ltsc-eval.yaml
/home/jose/.nido/registry/blueprints/windows-server-2022-core-eval.yaml
/home/jose/.nido/registry/blueprints/shared/windows-autounattend.xml
```

The stale `windows-11-iot-ltsc-eval.yaml` still had the old ISO and script mapping:

```yaml
iso_name: "windows-11-iot-ltsc-eval.iso"

scripts:
  autounattend.xml: "@shared/windows-autounattend.xml"
```

The fixed blueprint in the local Nido source checkout had:

```yaml
iso_name: "windows-11-iot-ltsc-eval-en-us.iso"

scripts:
  Autounattend.xml: "@shared/windows-autounattend.xml"
  windows-setup-openssh.ps1: "@shared/windows-setup-openssh.ps1"
```
The stale PAL-home registry caused `nido build windows-11-iot-ltsc-eval` to continue using the old media and old provisioning layout after the binary update.

## Expected Behavior

After `nido update`, Nido should make the installed runtime internally coherent.

For built-in Nido blueprint assets, one of these behaviors should happen:

- automatically migrate/sync bundled blueprint files into PAL home;
- or detect that PAL-home built-in blueprint files are stale and report an actionable warning;
- or provide a documented command such as `nido blueprint update` / `nido registry update` that updates built-in blueprint assets with backups.

The user should not have to manually copy files from a source checkout into:

```text
/home/jose/.nido/registry/blueprints
```

to make a newly updated Nido binary use its matching blueprint definitions.

## Impact

This creates a false-negative validation path.

The operator sees Nido successfully updated to `v4.5.23`, but Windows builds still fail with the pre-fix blueprint behavior because the PAL-home registry is stale.

This blocked downstream Windows installer validation for Needlex even after the Nido fix had been released locally.

## Manual Workaround Used

The PAL-home registry was manually backed up and updated from the local Nido source checkout:

```text
Backup:
/home/jose/.nido/registry/backups-20260602180901
```

Files copied from:

```text
/home/jose/hpdev/Libraries/nido/registry/blueprints
```

to:

```text
/home/jose/.nido/registry/blueprints
```

After copying, `nido build windows-11-iot-ltsc-eval` generated a seed directory containing:

```text
Autounattend.xml
```

which confirms the updated blueprint assets were being used.

## Suggested Fix Direction

Nido likely needs explicit registry/blueprint migration behavior as part of update or startup.

Recommended properties:

- treat PAL home as the source of runtime state, but distinguish user-authored blueprints from Nido-bundled managed blueprints;
- preserve local user changes by creating backups before overwriting managed blueprint assets;
- record a managed asset version or checksum so Nido can detect stale installed blueprints;
- surface clear diagnostics in `nido doctor` and `nido blueprint list/info` when the binary and installed managed blueprints are out of sync;
- avoid silently continuing with stale managed blueprints after a binary update that expects newer registry assets.

## Acceptance Criteria

The fix should allow this sequence to work without manual file copying:

```bash
/home/jose/.nido/bin/nido update
/home/jose/.nido/bin/nido blueprint info windows-11-iot-ltsc-eval
/home/jose/.nido/bin/nido build windows-11-iot-ltsc-eval --json
```

After update, `blueprint info` / the PAL-home YAML should reflect the fixed bundled blueprint definitions, including:

```yaml
iso_name: "windows-11-iot-ltsc-eval-en-us.iso"
scripts:
  Autounattend.xml: "@shared/windows-autounattend.xml"
  windows-setup-openssh.ps1: "@shared/windows-setup-openssh.ps1"
```
