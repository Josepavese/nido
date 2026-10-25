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
| AUR `nido-bin` | wired, activates with secrets | release | AUR account + SSH key | Upstream registration is closed with no manual queue; no account exists. After reopening: account/email verification, SSH public key and release secrets. Nothing for the maintainer to do now |
| WinGet | wired (`WINGET_TOKEN`) | after first merge | classic PAT | All 10 checks and CLA pass for 4.9.0; only a Microsoft moderator can merge [PR #460843](https://github.com/microsoft/winget-pkgs/pull/460843). No maintainer action now |
| Glama | auto-index + crawler | crawler | GitHub OAuth | Ownership claimed; hosted auto-release stays off. Support acknowledged the withdrawal ticket; completion remains pending. See `DISTRIBUTION.md` §1.6. Discovery and local setup only |
| mcpdir.dev / MCP Hub / PulseMCP / MCP Central | full (crawl) | crawler | none | none |
| awesome lists (mcp-servers, agent sandboxes, agent runtime, awesome-go) | submissions opened | – | GitHub account | **one-off:** shepherd the open PRs/issues |
| Manual directory forms (mcp.so, mcpservers.org, MCP Market, Product Hunt) | none | – | per-site accounts | **one-off:** submit with the content in `installers/README.md` |
| Smithery | removed | – | – | none (integration retired; no release publication) |

## 2. Recurring manual actions

| When | Action | Where the state lives |
| --- | --- | --- |
| Once, now | await the WinGet moderator and Glama support; register AUR only after reopening. Account verification is the remaining personal action, when available | links in `DISTRIBUTION.md` §2.7–§2.9 |
| Every release | inspect per-job results and read the published versions back; container/package smoke checks are non-blocking, so warnings need explicit reconciliation | Actions run for the tag |
| Quarterly | re-read `DISTRIBUTION.md` §3 (dropped channels), check PAT/secret expiry, confirm Pages and keyring are live, close stale submission PRs | this file + `DISTRIBUTION.md` |
| Before 2029-10-24 | rotate the package signing key: new key, new keyring file name, update `signed-by=`/`gpgkey=` and `PACKAGES_GPG_KEY`, keep the old keyring published for one cycle | `DISTRIBUTION.md` §2.10 |
| On GitHub plan changes | confirm Actions minutes/quota still cover macOS/Windows jobs; their three OS checks are required by branch protection | repo settings |

## 3. Release-day contract

Automatic: assets, GitHub Release, npm, MCP Registry, GHCR image, APT/DNF
repositories, Homebrew and Scoop cron sync, plus post-publish smoke checks.
Linux, macOS and Windows checks, the VM lifecycle smoke test and release
archive checks qualify the candidate before tagging. Nix CI must also pass
when its flake, Go inputs or changelog version change.

Some publication jobs use `continue-on-error`; a green workflow alone does not
prove publication. Inspect each configured job and read the published version
back. A missing credential means unconfigured, not published. A third-party
review or crawl remains pending until its result is observable.

Manual: only the items in §2. A release is complete when `nido version`,
`npm view nido-mcp version`, the MCP Registry listing, the container tag and
the signed [APT package index](https://josepavese.github.io/nido/apt/dists/stable/main/binary-amd64/Packages.gz) all agree on the
same version. Also verify the Homebrew/Scoop manifests and checksums, the
signed APT/DNF package metadata, and a local install from the published native
archive. Visually inspect that installed TUI at wide and compact sizes.
To repair APT/DNF, dispatch `release.yml` on current `main` with the existing
`version` tag and `packages_only=true`. This skips release creation and every
other publisher, verifies the native archives against the release checksums,
and rebuilds the signed repositories from those exact binaries.

When a channel disagrees, repair that channel without overwriting immutable
release assets. Record external pending submissions explicitly in the release
notes; they do not become a successful publication because CI continued.

## 4. Change control

Adding or dropping a channel is a documentation-led change in the same PR:
update this table, the channel table in `DISTRIBUTION.md`, the install commands
in `installers/README.md`, and the workflow job. A channel that cannot be
verified end-to-end is not listed as supported.
