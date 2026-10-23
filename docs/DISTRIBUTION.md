# Distribution & visibility

How Nido reaches users and AI-agent tooling, and what runs automatically on
every release. Research verified on **2026-10-23**; account/token requirements
are listed per channel.

The release workflow (`.github/workflows/release.yml`) builds the assets, creates
the GitHub Release, and then publishes to the channels below. A channel without
its secret is **skipped green**, never failed – so adding credentials is always
incremental.

---

## 1. Channels automated in this repository

| Channel | Mechanism | Auth | Secret / setting | Notes |
| --- | --- | --- | --- | --- |
| **GitHub Releases** | tarballs, `nido.mcpb`, `server.json`, `SHA256SUMS`, installers | `GITHUB_TOKEN` | none | Asset names are a contract for every other channel. |
| **Official MCP Registry** | `mcp-publisher login github-oidc` + `publish` | GitHub OIDC (`id-token: write`) | none | Already live: `io.github.Josepavese/nido`. Versioned, immutable releases. |
| **npm** `nido-mcp` | launcher in [`npm/`](../npm), downloads + verifies the release binary lazily | npm trusted publishing (OIDC) | repo variable `NPM_TRUSTED_PUBLISHER=true` + one npm account | No long-lived npm token. Bootstrap: first publish + Trusted Publisher config (see §3.1). |
| **WinGet** | `vedantmgoyal9/winget-releaser` updates the manifest | classic PAT (`public_repo`) | `WINGET_TOKEN` | First manifest must be merged manually; afterwards updates are automated. |
| **Scoop** (own bucket) | release job rewrites `bucket/nido.json` in `Josepavese/scoop-nido` | fine-grained PAT | `SCOOP_BUCKET_TOKEN` | The bucket's own Excavator (`distribution/scoop/`) also self-updates with zero secrets. |
| **AUR** `nido-bin` | PKGBUILD rendered from `distribution/aur/`, pushed by CI | AUR account + SSH key | `AUR_SSH_PRIVATE_KEY`, `AUR_USERNAME`, `AUR_EMAIL` | x86_64 only, `-bin` package. |
| **Smithery** | `smithery mcp publish nido.mcpb -n josepavese/nido` | Bearer API key | `SMITHERY_API_KEY` | Stdio artifact; Nido does not need to be hosted. |
| **Glama** | auto-index + `glama.json` claim | GitHub OAuth, one-time | none | `glama.json` is in the repo root. The claim is manual once. |
| **mcpdir.dev / MCP Hub** | crawls GitHub topic `mcp-server`, npm, official registry | none | none | Topics already set on the repo. |

Indirectly covered with no work: **PulseMCP** and **MCP Central** ingest the
official registry (PulseMCP submissions are paused since 2026-09-03), and the
GitHub MCP Registry syncs versions after a manual onboarding.

---

## 2. One-time setup checklist

### 2.1 GitHub repository (already partly done)

- Topics: `mcp-server`, `mcp`, `model-context-protocol`, `qemu`, `virtual-machine`,
  `ai-agents`, `golang` — set.
- `glama.json` at root — in repo; open the Glama server page and use **Claim**
  (GitHub OAuth) once.
- `.mcp.json` at root — Open Plugins detection (Cursor Directory and friends).

### 2.2 Homebrew tap (zero secrets)

Create the tap repository `Josepavese/homebrew-nido` and copy
`distribution/homebrew/templates/nido.rb` plus
`distribution/homebrew/.github/workflows/update-formula.yml` into it. The
workflow runs every 6 hours, renders the formula from the latest GitHub Release
and commits with the tap's own `GITHUB_TOKEN`.

```bash
brew install Josepavese/nido/nido
```

`homebrew-core` is **not** viable yet: its acceptance policy requires roughly
90 forks / 90 watchers / 225 stars for an owner self-submission.

### 2.3 Scoop bucket (zero secrets)

Create `Josepavese/scoop-nido` with `distribution/scoop/bucket/nido.json` and
`distribution/scoop/.github/workflows/excavator.yml`, then add the topic
`scoop-bucket`.

```powershell
scoop bucket add nido https://github.com/Josepavese/scoop-nido
scoop install nido
```

For immediate updates on release (instead of the 4-hour Excavator window), add a
fine-grained PAT with `contents: write` on the bucket as `SCOOP_BUCKET_TOKEN`.

### 2.4 npm (one account, one bootstrap, then OIDC)

1. Create an npm account (verify the username is available) and enable 2FA.
2. Publish the first version of `nido-mcp` (the `nido` name is taken by an
   unrelated 2015 package; `nido-mcp`, `nido-cli` and `@josepavese/nido` are free).
   Either `npm publish` manually from `npm/`, or `npm stage publish` + 2FA
   promotion.
3. On npmjs.com → package → Settings → Trusted publishing add:
   owner `Josepavese`, repository `nido`, workflow filename `release.yml`,
   and enable **Allow npm publish** (new configurations are stage-only by default).
4. Set the repository variable `NPM_TRUSTED_PUBLISHER=true`.

Note: unvalidated trusted-publisher configurations expire after ~2 days, so
complete one publish shortly after configuring. Trusted publishing requires the
GitHub-hosted runner and `id-token: write` (already set in the workflow).

### 2.5 WinGet

1. Fork `microsoft/winget-pkgs` as `Josepavese/winget-pkgs`.
2. Open the first manifest PR with the three files under
   `distribution/winget/manifests/j/Josepavese/Nido/4.7.0/` (update version,
   URL and SHA-256 for each release you submit). Merge times for a new portable
   package can be days; updates are much faster.
3. Create a **classic** PAT with `public_repo` (fine-grained tokens are not
   supported by the tooling) and store it as `WINGET_TOKEN`.

After the first version is merged, `winget upgrade Josepavese.Nido` follows every
release automatically.

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

### 2.7 Smithery

Log in to Smithery with GitHub, claim the `josepavese` namespace, create an API
key (Settings → API keys) and store it as `SMITHERY_API_KEY`. Do **not** use
service tokens: those have a 24-hour TTL. Then `smithery mcp publish` runs on
every release using the `nido.mcpb` asset.

### 2.8 Manual submissions (once, no automation possible)

| Where | What to submit | Cost |
| --- | --- | --- |
| `github.com/mcp` GitHub MCP Registry | request onboarding in `github/github-mcp-server` discussions with the registry name; versions sync afterwards | free |
| `punkpeye/awesome-mcp-servers` | PR adding one line to the right category; agents are fast-tracked by appending 🤖🤖🤖 to the PR title | free |
| `awesome-go` | PR after the next release makes pkg.go.dev reachable (required check) | free |
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
resolved to a broken `v1.0.0`. The module was renamed to
`github.com/Josepavese/nido/v4`; from the **next release** on:

```bash
go install github.com/Josepavese/nido/v4/cmd/nido@latest
```

and <https://pkg.go.dev/github.com/Josepavese/nido/v4> becomes the Go reference
page. Old `v1.x` tags remain historical.

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
URL must contain `mcp` — both registry requirements.

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
