# Distribution & visibility

How Nido reaches users and AI-agent tooling, and what runs automatically on
every release. Research verified on **2026-10-25**; the whole pipeline was then
validated end-to-end on `v4.8.1` (§1.1), `v4.8.3` (§1.3), the GHCR container
(§1.4) and the signed APT/DNF repositories (§1.5). Release-day manual actions
are governed by [DISTRIBUTION-GOVERNANCE.md](DISTRIBUTION-GOVERNANCE.md).
Account/token requirements are listed per channel. For user-facing commands,
see [installation methods and MCP catalogs](../installers/README.md).

The release workflow (`.github/workflows/release.yml`) builds the assets, creates
the GitHub Release, and then publishes to the channels below. A channel without
its secret is **skipped green**, never failed – so adding credentials is always
incremental.

---

## 1. Channels automated at every release

| Channel | Mechanism | Auth | Secret / setting | Notes |
| --- | --- | --- | --- | --- |
| **GitHub Releases** | tarballs, `nido.mcpb`, `server.json`, `SHA256SUMS`, installers | `GITHUB_TOKEN` | none | Asset names are a contract for every other channel. |
| **Container image** (GHCR) | `distribution/docker/Dockerfile` built from the release binaries and pushed to `ghcr.io/josepavese/nido` | `GITHUB_TOKEN` (`packages: write`) | none | Every tag publishes `:{version}`, `:{minor}` and `:latest`, bundling QEMU; meant for `/dev/kvm` hosts and MCP stdio use. First publish needs the package made public (`gh api` step in the job). |
| **APT / DNF repositories** (GitHub Pages) | `bin/publish-packages.sh` builds `.deb`/`.rpm` with nfpm, signs and indexes them; `publish-packages` pushes `gh-pages` | `GITHUB_TOKEN` (`contents: write`) | `PACKAGES_GPG_KEY` (stored in repo secrets) | Live: <https://josepavese.github.io/nido/> — signed APT `Release`/`InRelease`, signed RPM packages and DNF `repomd.xml`; keyring at `/nido-archive-keyring.asc`; both architectures per format. |
| **Nix flake** (in repo) | `flake.nix` builds from source with `buildGoModule` and wraps QEMU; `Nix flake` CI job builds it on flake/Go changes | none | none | Live: `nix run github:Josepavese/nido`; version read from `CHANGELOG.md`, so tags track automatically. Nixpkgs packaging stays out of scope (§3). |
| **Official MCP Registry** | `mcp-publisher login github-oidc` + `publish` | GitHub OIDC (`id-token: write`) | none | Live: [`io.github.Josepavese/nido`](https://registry.modelcontextprotocol.io/); every tag adds an immutable version. Runs **after** npm and waits for the npm version to appear (publish-time scanning is asynchronous, so the registry validation otherwise races the publish). |
| **npm** `nido-mcp` | launcher in [`npm/`](../npm), downloads + verifies the release binary lazily | npm trusted publishing (OIDC) | repo variable `NPM_TRUSTED_PUBLISHER=true` + npm trusted-publisher config | Live: [`nido-mcp`](https://www.npmjs.com/package/nido-mcp) with the `latest` dist-tag; every release publishes with provenance, no token stored (§2.4). |
| **WinGet** | `vedantmgoyal9/winget-releaser` updates the manifest | classic PAT (`public_repo`) | `WINGET_TOKEN` | `Josepavese.Nido` 4.8.6 is submitted and **fully validated** ([microsoft/winget-pkgs#460843](https://github.com/microsoft/winget-pkgs/pull/460843): 10/10 checks green, CLA signed, token configured); only community-moderator merge remains. Updates automate once merged. |
| **Scoop** (companion bucket) | [`Josepavese/scoop-nido`](https://github.com/Josepavese/scoop-nido) Excavator rewrites `bucket/nido.json` | `GITHUB_TOKEN` of the bucket | none | Live, zero secrets; polled every 4 hours; the dated verification below records tested versions. The release job can also push immediately when `SCOOP_BUCKET_TOKEN` exists. |
| **AUR** `nido-bin` | PKGBUILD rendered from `distribution/aur/`, pushed by CI | AUR account + SSH key | `AUR_SSH_PRIVATE_KEY`, `AUR_USERNAME`, `AUR_EMAIL` | **Blocked upstream:** AUR closed new registrations after the June 2026 takeover wave and captchas are still flaky; the job is wired and activates as soon as the three secrets exist. x86_64 only, `-bin` package. |
| **Homebrew** (companion tap) | [`Josepavese/homebrew-nido`](https://github.com/Josepavese/homebrew-nido) cron renders the formula | `GITHUB_TOKEN` of the tap | none | Live, zero secrets; polled every 6 hours; the dated verification below records tested versions. |
| **Glama** | directory indexing | GitHub OAuth, one-time | auto-release off | [Nido listing](https://glama.ai/mcp/servers/Josepavese/nido) provides discovery and local installation instructions. Hosted deployment is outside this channel's intended use; withdrawal of the previously published Glama release remains pending (§1.6). |
| **mcpdir.dev / MCP Hub** | crawls GitHub topic `mcp-server`, npm, official registry | none | none | Topics already set on the repo. |

Indirectly covered with no work: **PulseMCP** and **MCP Central** ingest the
official registry (PulseMCP submissions are paused since 2026-09-03), and the
GitHub MCP Registry syncs versions after a manual onboarding.

### 1.1 Verified on v4.8.1 (2026-10-23)

- `v4.8.1` release published with all 10 assets; release workflow green,
  including the `Resolve distribution channels` gate.
- `io.github.Josepavese/nido` lists version `4.8.1` (MCP Registry).
- `go install github.com/Josepavese/nido/v4/cmd/nido@v4.8.1` reports
  `Version v4.8.1`; `proxy.golang.org` serves `@latest` = `v4.8.1` and
  <https://pkg.go.dev/github.com/Josepavese/nido/v4> is live. New versions may
  need the pkg.go.dev **Request** button once, or a `go get`, before they appear.
- Homebrew tap rendered formula `4.8.1` with SHA-256 values identical to
  `SHA256SUMS`; Scoop bucket manifest points to `4.8.1` with the matching zip
  hash. Both updates ran from their own scheduled workflows with no secrets.
- Credential-gated jobs (npm, WinGet, Scoop push, AUR) were reported
  as **skipped**, never failed.
- Open submissions: [awesome-mcp-servers#17428](https://github.com/punkpeye/awesome-mcp-servers/pull/17428),
  [winget-pkgs#460843](https://github.com/microsoft/winget-pkgs/pull/460843) (validated),
  [github-mcp-server#3572](https://github.com/github/github-mcp-server/issues/3572)
  (GitHub MCP Registry curation request).

### 1.2 Verified on 2026-10-24 (npm channel + MCP metadata)

- `nido-mcp@4.8.1` live on npm: first version created through staged publishing
  and promoted on npmjs.com; installing the tarball and running `nido version`
  inside it reports `v4.8.1` (the launcher downloads and SHA-256-verifies the
  release binary).
- **npm trusted publishing validated on `v4.8.2`**: the release workflow
  published `nido-mcp@4.8.2` with provenance through GitHub OIDC, with no
  stored token. npm processes new releases asynchronously (~1 minute) before the
  version becomes installable, which is why the registry job waits for it.
- The npm package declares `mcpName: io.github.Josepavese/nido`, the identity the
  MCP Registry cross-checks before registering a server that advertises an npm
  package.
- Scheduled **Update Registry** fixed: Ubuntu numeric directories that answer 403
  now fall back to the codename directory; the workflow is green and its catalog
  PR auto-merged with `ubuntu: 26.10`.
- Metadata positioning: GitHub repository description + 20 topics, and
  `server.json` title/description/repository/icon plus an npm package entry, all
  describe Nido as an MCP-native VM manager.

### 1.3 Verified on v4.8.3 (2026-10-24)

- Release workflow green end-to-end: GitHub release with all 10 assets, npm,
  MCP Registry and distribution gate; credential-gated channels skipped green.
- `nido-mcp@4.8.3` was published **by the workflow through OIDC** with
  provenance (no stored token), declares `mcpName:
  io.github.Josepavese/nido`, is `latest` on npm, and
  `npx -y nido-mcp@4.8.3 version` reports `Version v4.8.3`.
- `io.github.Josepavese/nido` `4.8.3` is `isLatest` in the MCP Registry with the
  new title, description, `websiteUrl`, repository and icon, plus both the
  `mcpb` and `npm` packages.
- Registry ordering fixed: the registry job now waits for the npm version to
  appear (`3-attempt` publish retry) instead of racing npm's asynchronous
  publish-time processing.
- Homebrew tap and Scoop bucket re-rendered to `4.8.3` (tap hashes match
  `SHA256SUMS`) with the new MCP-native description, via their own zero-secret
  scheduled workflows.
- `v4.8.2` stays live on GitHub Releases and npm but has no MCP Registry entry:
  it predates `mcpName`, and npm versions are immutable (§2.4).

### 1.4 Container channel (2026-10-24)

The GHCR image was validated locally before wiring it into the release: the
Dockerfile was built against the same Linux binaries the release ships, then
`nido version`, a full `nido doctor` (data dirs, QEMU, `qemu-img`, built-in ISO
writer, KVM acceleration) and an MCP `initialize` + `tools/list` handshake were
run inside the container. `doctor` reports `accelerator=kvm cpu=host` only when
the host passes `/dev/kvm` and the matching group (`--group-add "$(stat -c %g
/dev/kvm)"`); without them the image falls back to TCG and reports the
acceleration check as failed. The publish job is part of the release workflow,
so the image appears on the next tag and `:latest` follows.

### 1.5 Verified on 2026-10-25 (APT/DNF repositories, signed)

The `gh-pages` branch serves APT and DNF repositories built from the `v4.8.7`
release binaries with nfpm (`nido_4.8.7_amd64.deb`, `nido_4.8.7_arm64.deb`,
`nido-4.8.7-1.x86_64.rpm`, `nido-4.8.7-1.aarch64.rpm`). Both were installed
end-to-end from the **live Pages URLs** in throwaway containers:
`apt-get install nido` on `debian:bookworm-slim` and `dnf install nido` on
`fedora:latest`, each then reporting `nido version v4.8.7`; the APT `Release`
file indexes `amd64` and `arm64`, and `createrepo_c` produced the DNF
`repodata`/`nido.repo`. The same build was replayed locally before publishing.

On 2026-10-25 the repositories were **signed** with the Nido package archive key
`DBB87CB63A94ABA50B33C7B919C6A95C9D280B64` (expires 2029-10-24): GPG verified
`Release.gpg`, `InRelease`, the RPM package signatures and `repomd.xml.asc`,
and signed installs were exercised in clean containers (APT via `signed-by`,
DNF with `gpgcheck=1` + `repo_gpgcheck=1`, key fetched from the published
keyring and imported automatically).

---

### 1.6 Glama directory and local MCP (2026-10-25)

Glama is a discovery channel for Nido. Ownership through `glama.json` remains
verified, and the listing description directs users to install Nido and QEMU
on their own computer, then configure their MCP client to launch `nido mcp`
over stdio. Alternatively, the client can launch `npx -y nido-mcp@latest mcp`.
See the [installation guide](../installers/README.md) and the exact
[client configuration](mcp.md#configure-a-client).

**Auto-release is disabled**, verified after reloading the
[Glama release settings](https://glama.ai/mcp/servers/Josepavese/nido/admin/releases).
Keep it off when publishing future Nido releases. Repository indexing and
ownership do not require maintaining a hosted runtime. `glama.json` declares
maintainers; it has no hosting-disable setting.

**Hosted withdrawal is pending:** Glama release 4.8.7, published during earlier
container qualification, still appears in the release list. The inspected
Releases, Dockerfile and Listing admin pages expose no withdrawal or hosting
disable control. Glama support must confirm how to retire that release and
disable hosted deployment while preserving the directory entry. Disabling
auto-release alone does not withdraw an existing release. Glama's generated
usage FAQ still suggests hosted deployment and also needs correction.

No persistent hosted instance was deployed during qualification, and no local
VM directories or credentials were supplied. A Glama-hosted process would
manage its own environment and could not access the user's local VM inventory.
Use the local stdio configuration to manage workstation VMs. The separate GHCR
image remains available for user-managed hosts (§1.4).

## 2. One-time setup checklist

### 2.1 GitHub repository (already partly done)

- About: local VM manager for AI agents, accessible through CLI, TUI and native MCP.
- Topics cover the actual product: VM management, AI agents, MCP, QEMU,
  virtualization, provisioning and development/testing. Keep the GitHub About
  settings and channel metadata focused on these capabilities; avoid generic
  buzzwords and unqualified performance or security claims.
- `glama.json` at root - in repo; Glama ownership is verified. Keep the listing
  and local installation instructions current, with auto-release off (§1.6).
- `.mcp.json` at root - Open Plugins detection (Cursor Directory and friends).

### 2.2 Homebrew tap - live, zero secrets

[`Josepavese/homebrew-nido`](https://github.com/Josepavese/homebrew-nido) exists
with `Formula/nido.rb`, `templates/nido.rb` and
`.github/workflows/update-formula.yml`. The workflow runs every 6 hours,
re-renders the formula from the latest GitHub Release and commits with the tap's
own `GITHUB_TOKEN`; no repository secret is involved. Verified with a manual
dispatch.

```bash
brew install Josepavese/nido/nido
```

`homebrew-core` is **not** viable yet: its acceptance policy requires roughly
90 forks / 90 watchers / 225 stars for an owner self-submission.

### 2.3 Scoop bucket - live, zero secrets

[`Josepavese/scoop-nido`](https://github.com/Josepavese/scoop-nido) exists with
`bucket/nido.json` and `.github/workflows/excavator.yml`; the topic
`scoop-bucket` is set. The Excavator checks GitHub Releases every 4 hours and
rewrites version/URL/hash with the bucket's own `GITHUB_TOKEN`.

```powershell
scoop bucket add nido https://github.com/Josepavese/scoop-nido
scoop install nido
```

For immediate updates on release (instead of the 4-hour window), add a
fine-grained PAT with `contents: write` on the bucket as `SCOOP_BUCKET_TOKEN`;
the release job then pushes the manifest commit directly.

### 2.4 npm - live and fully automatic since 2026-10-24

`nido-mcp` is on npm (`latest`, public, maintained by `josepavese`). The `nido`
name is taken by an unrelated 2015 package; `nido-mcp` belongs to this project.
The first version was created with `npm stage publish` (no 2FA needed) and
promoted on npmjs.com with security-key 2FA, because a direct `npm publish` from
this account requires a TOTP that a security-key-only account cannot produce.

Publishing is now tokenless: the npm **Trusted Publisher** points at
`Josepavese/nido`, workflow `release.yml`, with **npm publish** allowed, and the
repository variable `NPM_TRUSTED_PUBLISHER=true` enables the job. `v4.8.2` was
the first release published this way (OIDC, with provenance).

Two operational details:

- npm processes new releases **asynchronously** (publish-time scanning): `npm
  publish` returns success while `registry.npmjs.org/nido-mcp/<version>` is still
  404 for up to a minute. The MCP Registry validates that the version exists, so
  its job polls first and only then registers the server.
- The npm package must declare `mcpName: io.github.Josepavese/nido` (matching
  `server.json.name`). The MCP Registry rejects npm packages without it, and npm
  versions are immutable, so a version published without the field can never be
  registered: `v4.8.3` carries the fix.

**No token is stored anywhere:** trusted publishing uses GitHub OIDC, so the
only npm-related setting in GitHub is the repository variable. GitHub Actions
secrets are scoped to a single repository (a personal account has no
organization-wide secrets), which is why every credential-gated channel gets its
own secret here.

### 2.5 WinGet

1. The fork `Josepavese/winget-pkgs` exists and the manifest PR carries
   `Josepavese.Nido` 4.8.6 with the **1.12.0** manifest schema:
   [microsoft/winget-pkgs#460843](https://github.com/microsoft/winget-pkgs/pull/460843).
   The Microsoft CLA is signed and all ten validation checks are green
   (`Validation-Completed`, `Azure-Pipeline-Passed`); only community-moderator
   approval remains, which can take days for a new portable package. An earlier
   submission ([#460549](https://github.com/microsoft/winget-pkgs/pull/460549))
   was closed as empty after a bad rebase and is superseded.
2. `WINGET_TOKEN` is configured: a **classic** PAT with `public_repo`
   (fine-grained tokens are not supported by the tooling) that lets
   `winget-releaser` push the fork branch and open the update PR on every
   release. Rotate it before it expires.

After the first version is merged, `winget upgrade Josepavese.Nido` follows every
release automatically. The in-repo template lives under
`distribution/winget/manifests/j/Josepavese/Nido/4.8.6/`.

### 2.6 AUR

1. Create an AUR account at `aur.archlinux.org` and add an SSH public key.
2. Create the `nido-bin` package once:
   ```bash
   git clone ssh://aur@aur.archlinux.org/nido-bin.git
   cp PKGBUILD .   # render from distribution/aur/PKGBUILD.in
   makepkg --printsrcinfo > .SRCINFO
   git add PKGBUILD .SRCINFO && git commit -m "initial" && git push
   ```
3. Store the private key as `AUR_SSH_PRIVATE_KEY`, plus `AUR_USERNAME` and
   `AUR_EMAIL` (optional, defaults exist).

AUR explicitly warns that malfunctioning automated updates can get accounts and
packages removed; the job is `continue-on-error`, so a failure never blocks a
release.

### 2.7 Smithery - removed

The `SMITHERY_API_KEY` channel was dropped: Smithery shut down its MCP proxy and
managed identity on 2026-10-31, so the account and key no longer bought durable
reach. The release job, the gate output and the secret entry are gone; no action
is required from maintainers. Nido remains discoverable through the Official
MCP Registry and the directories listed in §2.8.

### 2.8 Manual submissions (once, no automation possible)

| Where | What to submit | Cost |
| --- | --- | --- |
| `github.com/mcp` GitHub MCP Registry | curation request open: [github/github-mcp-server#3572](https://github.com/github/github-mcp-server/issues/3572); versions sync afterwards | free |
| `punkpeye/awesome-mcp-servers` | PR open: [#17428](https://github.com/punkpeye/awesome-mcp-servers/pull/17428) (Code Execution, 🤖🤖🤖 fast-track) | free |
| `awesome-go` | eligible now that pkg.go.dev and `go install` work; not submitted yet | free |
| `mcp.so` | repository URL via form | free with review / $39 fast-track |
| `mcpservers.org` (= wong2 list) | form, include `io.github.Josepavese/nido` in "Official MCP Registry Name" | free 2 weeks / $39 24h |
| MCP Market (`mcpmarket.com`) | form | free 4–6 weeks / $29 24h |
| `msyvr/awesome-agent-sandboxes` | PR open: [#79](https://github.com/msyvr/awesome-agent-sandboxes/pull/79) (Standalone / Local Tools, KVM tier; `data/sandboxes.yaml` + regenerated artifacts) | free |
| `sandbaseai/awesome-agent-runtime` | issue open: [#31](https://github.com/sandbaseai/awesome-agent-runtime/issues/31) (Execution Sandbox category) | free |
| Product Hunt / DevHunt / TAAFT | launch | free / $19 / free |

Newsletters (Console.dev, Golang Weekly, Ben's Bites) are editorial pitches, not
automated channels.

### 2.9 Container image (GHCR)

Nothing to configure: the publish job in `.github/workflows/release.yml` builds
`distribution/docker/Dockerfile` from the release binaries and pushes
`ghcr.io/josepavese/nido` with `GITHUB_TOKEN` (`packages: write`). The first
publish also needs the package to be made public: the job runs
`gh api -X PATCH /user/packages/container/nido -f visibility=public` with
`continue-on-error`, so a permission hiccup cannot fail the release – check
<https://github.com/Josepavese?tab=packages> after the first tag and flip the
visibility by hand if the API call was skipped.

### 2.10 APT / DNF repositories (GitHub Pages)

Pages serves the `gh-pages` branch at <https://josepavese.github.io/nido/> and
the `publish-packages` job rebuilds the repositories on every tag with the
workflow token (`contents: write`). It rewrites `gh-pages` as a single commit so
the package history does not grow the repository, keeps the last
`KEEP_VERSIONS` (default 5) releases in the pools, and tolerates "already
enabled" when ensuring Pages.

Signing uses the **`PACKAGES_GPG_KEY`** repository secret (ASCII-armored private
key, no passphrase, 4096-bit RSA, expires 2029-10-24). The public half is
committed at `distribution/packages/nido-archive-keyring.asc` and served as
`/nido-archive-keyring.asc`; clients trust it through `signed-by=` (APT) or the
`gpgkey=` line in `nido.repo` (DNF). The workflow signs the APT
`Release`/`InRelease`, every RPM package (`rpmsign`) and the DNF
`repomd.xml.asc`. **Rotation:** generate a new key, publish it under a new
keyring file name (e.g. `nido-archive-keyring-2029.asc`), update `signed-by=`/
`gpgkey=` and the `PACKAGES_GPG_KEY` secret together, and keep the old keyring
published for one release cycle so existing clients can upgrade.

Without the secret the script falls back to unsigned repositories
(`trusted=yes` / `gpgcheck=0`), which keeps local builds and forks working; see
[installers](../installers/README.md#apt-debian--ubuntu).

---

## 3. Intentionally not automated

| Channel | Why |
| --- | --- |
| Flathub | rejects console software; explicit "no AI agent PRs" policy. |
| Homebrew core | popularity thresholds (see above). |
| Chocolatey | every version is human-moderated until the package becomes "trusted"; not worth it early. |
| Snap Store | `classic` confinement needs an uncertain human review; packaging QEMU inside the snap is heavy. |
| Nixpkgs | one PR + weeks of review; automatic bumps only after merge. The in-repo flake (see §1) covers `nix run` without that review queue. |
| Docker MCP Catalog | the catalog runs servers without host `/dev/kvm`, which a VM manager needs; the GHCR image is for self-hosting instead. |
| Anthropic Claude directory | paid Claude plan; MCPB desktop extensions are no longer accepted (plugin bundles only). |
| OpenAI plugin directory | local/desktop path is limited to allowlisted partners. |
| Docker Hub | GHCR already covers the zero-credential case; a Docker Hub mirror would need an account for little extra reach. |
| LobeHub / Cursor Directory / mcp.so / MCP Market / mcpservers.org | browser-only submission or review; listing is one-shot, version updates are not supported via CI. |

---

## 4. Go module consumers (`go install`, pkg.go.dev)

Because the module path was `github.com/Josepavese/nido` while releases were
tagged `v4.x`, the Go proxy rejected those versions and `go install …@latest`
resolved to a broken `v1.0.0`. The module is now
`github.com/Josepavese/nido/v4` and the path is verified on `v4.8.1`:

```bash
go install github.com/Josepavese/nido/v4/cmd/nido@latest   # reports the installed version
```

<https://pkg.go.dev/github.com/Josepavese/nido/v4> is the Go reference page; a
fresh tag may need one **Request** click on pkg.go.dev before it is generated.
Old `v1.x` tags remain historical.

---

## 5. Release asset contract

Every packaging channel assumes these names, produced by `bin/build-release.sh`:

```
nido-linux-amd64.tar.gz      nido-darwin-amd64.tar.gz
nido-linux-arm64.tar.gz      nido-darwin-arm64.tar.gz
nido-windows-amd64.zip       nido.mcpb        server.json
install.sh                   install.ps1      SHA256SUMS
```

`server.json` is generated per release with the real version, asset URL and
`fileSha256` of `nido.mcpb` (the repository copy keeps `0.0.0-dev` placeholders).
`description` in `server.json` must stay ≤ 100 characters and the MCPB artifact
URL must contain `mcp` - both registry requirements.

---

## 6. Sources

- Official MCP Registry: <https://modelcontextprotocol.io/registry/github-actions>, <https://github.com/modelcontextprotocol/registry>
- MCP package types (mcpb rules): <https://github.com/modelcontextprotocol/registry/blob/main/docs/modelcontextprotocol-io/package-types.mdx>
- npm trusted publishing: <https://docs.npmjs.com/trusted-publishers>; staged publishing: <https://github.blog/changelog/2026-10-02-npm-staged-publishing-now-supports-creating-new-packages/>
- npm malware scanning / dual-use: <https://github.blog/changelog/2026-07-28-npm-publish-time-malware-scanning-and-dual-use-metadata/>
- Glama claim: <https://glama.ai/blog/2025-07-08-what-is-glamajson>
- Glama metadata schema: <https://glama.ai/mcp/schemas/server.json>
- awesome-mcp-servers agent fast-track: <https://github.com/punkpeye/awesome-mcp-servers/blob/main/CONTRIBUTING.md>
- WinGet manifests / validation: <https://github.com/microsoft/winget-pkgs/blob/master/doc/manifest/schema/1.10.0/installer.md>, <https://github.com/microsoft/winget-pkgs/blob/master/doc/Validation.md>
- winget-releaser: <https://github.com/vedantmgoyal9/winget-releaser>
- Scoop autoupdate / GithubActions: <https://github.com/ScoopInstaller/Scoop/wiki/App-Manifest-Autoupdate>, <https://github.com/ScoopInstaller/GithubActions>
- Homebrew taps / acceptance policy: <https://github.com/Homebrew/brew/blob/main/docs/How-to-Create-and-Maintain-a-Tap.md>, <https://github.com/Homebrew/brew/blob/main/docs/Package-Acceptance-Policy.md>
- AUR guidelines: <https://wiki.archlinux.org/title/AUR_submission_guidelines>; AUR action: <https://github.com/KSXGitHub/github-actions-deploy-aur>
- Go modules major versions: <https://go.dev/ref/mod#major-version-suffixes>
