# Distribution governance

How much of Nido's distribution runs itself, which actions genuinely require a
human, and when to revisit them. The channel-by-channel mechanics live in
[`DISTRIBUTION.md`](DISTRIBUTION.md); this page is the small operating contract
on top of it.

The principle: **automation owns repetition, humans own identity.** Anything
that only proves ownership (merging a first PR, claiming a profile, signing up
to an external account, rotating a credential) stays manual; everything that
repeats on every tag runs in the release workflow or in the companion
repositories' cron jobs.

## 1. Automation map

| Channel | Automation | Trigger | Credential / account | Manual action left |
| --- | --- | --- | --- | --- |
| GitHub Releases | full | `v*` tag | `GITHUB_TOKEN` | none |
| npm `nido-mcp` | full (OIDC trusted publisher) | release | trusted-publisher setting | rotate nothing; revisit npm 2FA policy when it changes |
| Official MCP Registry | full (OIDC) | release | none | none |
| GHCR container | full | release | `GITHUB_TOKEN` | once: package visibility (handled by job, admin repair only if it fails) |
| APT / DNF repositories | full (build, sign, index, Pages) | release | `PACKAGES_GPG_KEY` secret | key rotation (see §2) |
| Homebrew tap | full (cron) | 6 h poll | none | none |
| Nix flake | full CI build on flake/Go changes | push / PR | none | none |
| Scoop bucket | full (cron) | 4 h poll | none | none |
| AUR `nido-bin` | wired, activates with secrets | release | AUR account + SSH key | **one-off:** register when AUR reopens, then add 3 secrets |
| WinGet | wired (`WINGET_TOKEN`) | after first merge | classic PAT | **one-off:** Microsoft moderator merges [PR #460843](https://github.com/microsoft/winget-pkgs/pull/460843) |
| Glama | auto-index + crawler | crawler | GitHub OAuth | **one-off:** claim profile; press re-sync when releases lag. Promotion only — Nido is local and Glama never hosts the MCP server |
| mcpdir.dev / MCP Hub / PulseMCP / MCP Central | full (crawl) | crawler | none | none |
| awesome lists (mcp-servers, agent sandboxes, agent runtime, awesome-go) | submissions opened | – | GitHub account | **one-off:** shepherd the open PRs/issues |
| Manual directory forms (mcp.so, mcpservers.org, MCP Market, Product Hunt) | none | – | per-site accounts | **one-off:** submit with the content in `installers/README.md` |
| Smithery | removed | – | – | none (platform shut down 2026-10-31) |

## 2. Recurring manual actions

| When | Action | Where the state lives |
| --- | --- | --- |
| Once, now | merge the WinGet initial PR; claim Glama; submit awesome-list/directory forms; register AUR when it reopens | links in `DISTRIBUTION.md` §2.7–§2.9 |
| Every release (2 min, only if something looks off) | glance at the release workflow summary; the container and package jobs smoke-test their own published URLs and log a warning instead of failing the release | Actions run for the tag |
| Quarterly | re-read `DISTRIBUTION.md` §3 (dropped channels), check PAT/secret expiry, confirm Pages and keyring are live, close stale submission PRs | this file + `DISTRIBUTION.md` |
| Before 2029-10-24 | rotate the package signing key: new key, new keyring file name, update `signed-by=`/`gpgkey=` and `PACKAGES_GPG_KEY`, keep the old keyring published for one cycle | `DISTRIBUTION.md` §2.10 |
| On GitHub plan changes | confirm Actions minutes/quota still cover macOS/Windows jobs; they are the slowest but not blocking | repo settings |

## 3. Release-day contract

Automatic: assets, GitHub Release, npm, MCP Registry, GHCR image, APT/DNF
repositories, Homebrew and Scoop cron sync, plus post-publish smoke checks.
The workflow is designed so a third-party hiccup is skipped green or warns; a
release is never failed by a distribution channel.

Manual: only the items in §2. A release is complete when `nido version`,
`npm view nido-mcp version`, the MCP Registry listing, the container tag and
`https://josepavese.github.io/nido/apt/dists/stable/Release` all agree on the
same version. When they do not, fix the channel, not the release.

## 4. Change control

Adding or dropping a channel is a documentation-led change in the same PR:
update this table, the channel table in `DISTRIBUTION.md`, the install commands
in `installers/README.md`, and the workflow job. A channel that cannot be
verified end-to-end is not listed as supported.
