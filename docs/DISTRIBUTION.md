# Distribution & visibility

How Nido reaches users and AI-agent tooling, and what runs automatically on
every release. Research verified on **2026-10-23**; the whole pipeline was then
validated end-to-end on the `v4.8.1` tag (see §1.1). Account/token requirements
are listed per channel.

The release workflow (`.github/workflows/release.yml`) builds the assets, creates
the GitHub Release, and then publishes to the channels below. A channel without
its secret is **skipped green**, never failed – so adding credentials is always
incremental.

---

## 1. Channels automated at every release

| Channel | Mechanism | Auth | Secret / setting | Notes |
| --- | --- | --- | --- | --- |
| **GitHub Releases** | tarballs, `nido.mcpb`, `server.json`, `SHA256SUMS`, installers | `GITHUB_TOKEN` | none | Asset names are a contract for every other channel. |
| **Official MCP Registry** | `mcp-publisher login github-oidc` + `publish` | GitHub OIDC (`id-token: write`) | none | Live: `io.github.Josepavese/nido` through `4.8.1`; every tag adds an immutable version. Runs **after** npm (the registry validates that the advertised npm version exists). |
| **npm** `nido-mcp` | launcher in [`npm/`](../npm), downloads + verifies the release binary lazily | npm trusted publishing (OIDC) | repo variable `NPM_TRUSTED_PUBLISHER=true` + npm trusted-publisher config | Live: `nido-mcp@4.8.1` published 2026-10-24 (staged → promoted). Every tag publishes once the OIDC switch is on (§2.4). |
| **WinGet** | `vedantmgoyal9/winget-releaser` updates the manifest | classic PAT (`public_repo`) | `WINGET_TOKEN` | First manifest PR open: [microsoft/winget-pkgs#460549](https://github.com/microsoft/winget-pkgs/pull/460549) (needs the Microsoft CLA). Updates automate once merged + token exists. |
| **Scoop** (companion bucket) | [`Josepavese/scoop-nido`](https://github.com/Josepavese/scoop-nido) Excavator rewrites `bucket/nido.json` | `GITHUB_TOKEN` of the bucket | none | Live, zero secrets; polled every 4 hours, verified updating to `4.8.1`. The release job can also push immediately when `SCOOP_BUCKET_TOKEN` exists. |
| **AUR** `nido-bin` | PKGBUILD rendered from `distribution/aur/`, pushed by CI | AUR account + SSH key | `AUR_SSH_PRIVATE_KEY`, `AUR_USERNAME`, `AUR_EMAIL` | Not configured yet; x86_64 only, `-bin` package. |
| **Smithery** | `smithery mcp publish nido.mcpb -n josepavese/nido` | Bearer API key | `SMITHERY_API_KEY` | Not configured yet; stdio artifact, no hosting needed. |
| **Homebrew** (companion tap) | [`Josepavese/homebrew-nido`](https://github.com/Josepavese/homebrew-nido) cron renders the formula | `GITHUB_TOKEN` of the tap | none | Live, zero secrets; polled every 6 hours, verified updating to `4.8.1`. |
| **Glama** | auto-index + `glama.json` claim | GitHub OAuth, one-time | none | `glama.json` is in the repo root. The claim is manual once. |
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
- Credential-gated jobs (npm, WinGet, Scoop push, AUR, Smithery) were reported
  as **skipped**, never failed.
- Open submissions: [awesome-mcp-servers#17428](https://github.com/punkpeye/awesome-mcp-servers/pull/17428),
  [winget-pkgs#460549](https://github.com/microsoft/winget-pkgs/pull/460549),
  [github-mcp-server#3572](https://github.com/github/github-mcp-server/issues/3572)
  (GitHub MCP Registry curation request).

### 1.2 Verified on 2026-10-24 (npm channel + MCP metadata)

- `nido-mcp@4.8.1` live on npm: first version created through staged publishing
  and promoted on npmjs.com; installing the tarball and running `nido version`
  inside it reports `v4.8.1` (the launcher downloads and SHA-256-verifies the
  release binary).
- Scheduled **Update Registry** fixed: Ubuntu numeric directories that answer 403
  now fall back to the codename directory; the workflow is green and its catalog
  PR auto-merged with `ubuntu: 26.10`.
- Metadata positioning: GitHub repository description + 20 topics, and
  `server.json` title/description/repository/icon plus an npm package entry, all
  describe Nido as an MCP-native VM manager.

---

## 2. One-time setup checklist

### 2.1 GitHub repository (already partly done)

- Topics: `mcp`, `mcp-server`, `model-context-protocol`, `vm`, `vm-manager`,
  `virtual-machine`, `virtualization`, `qemu`, `kvm`, `sandbox`, `agent-sandbox`,
  `ai`, `ai-agents`, `agentic-workflows`, `llm`, `cli`, `tui`, `golang`, `linux`,
  `local-development` - set (20 topics, the GitHub maximum).
- `glama.json` at root - in repo; open the Glama server page and use **Claim**
  (GitHub OAuth) once.
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

### 2.4 npm - live since 2026-10-24

`nido-mcp@4.8.1` is on npm (`latest`, public, maintained by `josepavese`). The
`nido` name is taken by an unrelated 2015 package; `nido-mcp` belongs to this
project. The first version was created with `npm stage publish` (no 2FA needed)
and promoted on npmjs.com with security-key 2FA; a later `npm publish` uses
OIDC, so no npm token is stored anywhere.

Remaining one-time switch to fully automatic publishes (~2 minutes):

1. On npmjs.com → `nido-mcp` → Settings → **Trusted Publisher** → GitHub
   Actions: owner `Josepavese`, repository `nido`, workflow filename
   `release.yml`, and enable **Allow npm publish** (new configurations default
   to stage-only).
2. Set the repository variable `NPM_TRUSTED_PUBLISHER=true`.

From then on every `v*` tag publishes `nido-mcp` at the tag version with
provenance. Unvalidated trusted-publisher configurations expire after ~2 days,
so validate with the next release.

**No token is stored anywhere:** trusted publishing uses GitHub OIDC, so the
only npm-related setting in GitHub is the repository variable. GitHub Actions
secrets are scoped to a single repository (a personal account has no
organization-wide secrets), which is why every credential-gated channel gets its
own secret here.

### 2.5 WinGet

1. The fork `Josepavese/winget-pkgs` exists and the first manifest PR is open:
   [microsoft/winget-pkgs#460549](https://github.com/microsoft/winget-pkgs/pull/460549)
   (`Josepavese.Nido` 4.8.1, validated against the 1.10.0 schemas). Sign the
   Microsoft CLA through the bot comment to unblock CI; merge times for a new
   portable package can be days.
2. Create a **classic** PAT with `public_repo` (fine-grained tokens are not
   supported by the tooling) and store it as `WINGET_TOKEN`.

After the first version is merged, `winget upgrade Josepavese.Nido` follows every
release automatically. The in-repo template lives under
`distribution/winget/manifests/j/Josepavese/Nido/`.

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

### 2.7 Smithery - not configured yet

Log in to Smithery with GitHub, claim the `josepavese` namespace, create an API
key (Settings → API keys) and store it as `SMITHERY_API_KEY`. Do **not** use
service tokens: those have a 24-hour TTL. Then `smithery mcp publish` runs on
every release using the `nido.mcpb` asset.

### 2.8 Manual submissions (once, no automation possible)

| Where | What to submit | Cost |
| --- | --- | --- |
| `github.com/mcp` GitHub MCP Registry | curation request open: [github/github-mcp-server#3572](https://github.com/github/github-mcp-server/issues/3572); versions sync afterwards | free |
| `punkpeye/awesome-mcp-servers` | PR open: [#17428](https://github.com/punkpeye/awesome-mcp-servers/pull/17428) (Code Execution, 🤖🤖🤖 fast-track) | free |
| `awesome-go` | eligible now that pkg.go.dev and `go install` work; not submitted yet | free |
| `mcp.so` | repository URL via form | free with review / $39 fast-track |
| `mcpservers.org` (= wong2 list) | form, include `io.github.Josepavese/nido` in "Official MCP Registry Name" | free 2 weeks / $39 24h |
| MCP Market (`mcpmarket.com`) | form | free 4–6 weeks / $29 24h |
| Directory of agent sandboxes lists (`msyvr/awesome-agent-sandboxes`, `sandbaseai/awesome-agent-runtime`) | PR/issue | free |
| Product Hunt / DevHunt / TAAFT | launch | free / $19 / free |

Newsletters (Console.dev, Golang Weekly, Ben's Bites) are editorial pitches, not
automated channels.

---

## 3. Intentionally not automated

| Channel | Why |
| --- | --- |
| Flathub | rejects console software; explicit "no AI agent PRs" policy. |
| Homebrew core | popularity thresholds (see above). |
| Chocolatey | every version is human-moderated until the package becomes "trusted"; not worth it early. |
| Snap Store | `classic` confinement needs an uncertain human review; packaging QEMU inside the snap is heavy. |
| Nixpkgs | one PR + weeks of review; automatic bumps only after merge. A flake in-repo (`nix run github:Josepavese/nido`) is a cheaper alternative. |
| APT/RPM repositories on GitHub Pages | viable and zero-account (nfpm + `gh-pages`), but Pages is disabled and it needs an indexing/signing pass. Planned follow-up. |
| Docker MCP Catalog | needs a Dockerfile + image; a KVM manager cannot run meaningfully in the catalog's container model. |
| Anthropic Claude directory | paid Claude plan; MCPB desktop extensions are no longer accepted (plugin bundles only). |
| OpenAI plugin directory | local/desktop path is limited to allowlisted partners. |
| Docker Hub | GHCR would be the zero-credential alternative; image value is limited for a host-level VM manager. |
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
- Smithery publish: <https://smithery.ai/docs/build/publish>, <https://smithery.ai/docs/api-reference/servers/publish-a-server>
- Glama claim: <https://glama.ai/blog/2025-07-08-what-is-glamajson>
- awesome-mcp-servers agent fast-track: <https://github.com/punkpeye/awesome-mcp-servers/blob/main/CONTRIBUTING.md>
- WinGet manifests / validation: <https://github.com/microsoft/winget-pkgs/blob/master/doc/manifest/schema/1.10.0/installer.md>, <https://github.com/microsoft/winget-pkgs/blob/master/doc/Validation.md>
- winget-releaser: <https://github.com/vedantmgoyal9/winget-releaser>
- Scoop autoupdate / GithubActions: <https://github.com/ScoopInstaller/Scoop/wiki/App-Manifest-Autoupdate>, <https://github.com/ScoopInstaller/GithubActions>
- Homebrew taps / acceptance policy: <https://github.com/Homebrew/brew/blob/main/docs/How-to-Create-and-Maintain-a-Tap.md>, <https://github.com/Homebrew/brew/blob/main/docs/Package-Acceptance-Policy.md>
- AUR guidelines: <https://wiki.archlinux.org/title/AUR_submission_guidelines>; AUR action: <https://github.com/KSXGitHub/github-actions-deploy-aur>
- Go modules major versions: <https://go.dev/ref/mod#major-version-suffixes>
