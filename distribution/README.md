# Distribution templates

Ready-to-copy packaging assets for the channels described in
[`docs/DISTRIBUTION.md`](../docs/DISTRIBUTION.md).

| Directory | Destination | Secrets |
| --- | --- | --- |
| `homebrew/` | `Josepavese/homebrew-nido` (tap repo) | none (cron uses the tap's own `GITHUB_TOKEN`) |
| `scoop/` | `Josepavese/scoop-nido` (bucket repo) | none (Excavator in the bucket) or `SCOOP_BUCKET_TOKEN` for immediate updates |
| `aur/` | PKGBUILD is rendered by `release.yml` and pushed by CI | `AUR_SSH_PRIVATE_KEY`, `AUR_USERNAME`, `AUR_EMAIL` |
| `winget/` | first manifest PR to `microsoft/winget-pkgs` | `WINGET_TOKEN` (classic PAT, `public_repo`) for later automated updates |

The npm launcher lives in [`../npm`](../npm) and is published from
`release.yml` with npm trusted publishing (OIDC, no long-lived token).

The release asset names are a contract: `nido-<os>-<arch>.tar.gz`,
`nido-windows-amd64.zip`, `nido.mcpb`, `server.json`, `SHA256SUMS`.
Changing them silently breaks every channel below.
