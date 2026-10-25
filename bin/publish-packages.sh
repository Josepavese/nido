#!/usr/bin/env bash
# Build the APT and DNF repositories that GitHub Pages serves from the
# gh-pages branch. Release workflow job `publish-packages` runs this on every
# tag; it can also be run locally:
#
#   bin/publish-packages.sh v4.8.7
#
# Environment:
#   NFPM                nfpm binary (default: nfpm on PATH)
#   SITE_DIR            output tree for the site (default: build/packages-site)
#   DIST_DIR            where release tarballs are downloaded (default: build/packages-dist)
#   WORK_DIR            scratch directory (default: build/packages)
#   PREVIOUS_SITE_DIR   existing gh-pages checkout: keeps older versions around
#   KEEP_VERSIONS       how many versions to retain (default: 5)
#   PACKAGES_GPG_KEY    ASCII-armored private key: signs APT Release/InRelease,
#                       the RPM packages and the DNF repomd.xml
#
# The repositories are signed when PACKAGES_GPG_KEY is set; the public keyring
# is published as /nido-archive-keyring.asc. Without the secret they are
# unsigned, so APT needs `[trusted=yes]` and DNF `gpgcheck=0`.

set -euo pipefail

VERSION_INPUT=${1:-}
if [ -z "$VERSION_INPUT" ]; then
  echo "usage: $0 <version, e.g. v4.8.7>" >&2
  exit 2
fi
TAG="v${VERSION_INPUT#v}"
VERSION="${TAG#v}"

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
NFPM=${NFPM:-nfpm}
SITE_DIR=${SITE_DIR:-"$ROOT/build/packages-site"}
DIST_DIR=${DIST_DIR:-"$ROOT/build/packages-dist"}
WORK_DIR=${WORK_DIR:-"$ROOT/build/packages"}
PREVIOUS_SITE_DIR=${PREVIOUS_SITE_DIR:-}
KEEP_VERSIONS=${KEEP_VERSIONS:-5}
SIGNED=0
GPG_FINGERPRINT=""

for tool in "$NFPM" apt-ftparchive gzip python3; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "missing required tool: $tool" >&2
    exit 1
  fi
done
if ! command -v createrepo_c >/dev/null 2>&1; then
  echo "missing required tool: createrepo_c (apt install createrepo-c)" >&2
  exit 1
fi

rm -rf "$SITE_DIR" "$WORK_DIR"
mkdir -p "$SITE_DIR" "$DIST_DIR" "$WORK_DIR"

if [ -n "${PACKAGES_GPG_KEY:-}" ]; then
  command -v gpg >/dev/null 2>&1 || { echo "missing required tool for signing: gpg" >&2; exit 1; }
  echo "==> Importing the package signing key"
  export GNUPGHOME="$WORK_DIR/gnupg"
  mkdir -p "$GNUPGHOME"
  chmod 700 "$GNUPGHOME"
  printf '%s' "$PACKAGES_GPG_KEY" | gpg --batch --import
  GPG_FINGERPRINT=$(gpg --list-keys --with-colons | awk -F: '/^fpr/{print $10; exit}')
  SIGNED=1
  gpg --armor --export > "$SITE_DIR/nido-archive-keyring.asc"
fi

echo "==> Downloading release assets for $TAG"
for arch in amd64 arm64; do
  asset="nido-linux-$arch.tar.gz"
  if [ ! -f "$DIST_DIR/$asset" ]; then
    gh release download "$TAG" --pattern "$asset" --dir "$DIST_DIR"
  fi
  mkdir -p "$WORK_DIR/$arch"
  tar -xzf "$DIST_DIR/$asset" -C "$WORK_DIR/$arch" --strip-components=1
done

echo "==> Building .deb and .rpm packages for $VERSION"
DEB_DIR="$WORK_DIR/deb"
RPM_DIR="$WORK_DIR/rpm"
mkdir -p "$DEB_DIR" "$RPM_DIR"
for arch in amd64 arm64; do
  sed -e "s|__VERSION__|$VERSION|g" \
      -e "s|__ARCH__|$arch|g" \
      -e "s|__BIN_DIR__|$WORK_DIR/$arch|g" \
    "$ROOT/distribution/packages/nfpm.yaml.in" > "$WORK_DIR/nfpm-$arch.yaml"
  "$NFPM" package --config "$WORK_DIR/nfpm-$arch.yaml" --packager deb --target "$DEB_DIR"
  "$NFPM" package --config "$WORK_DIR/nfpm-$arch.yaml" --packager rpm --target "$RPM_DIR"
done
ls -l "$DEB_DIR" "$RPM_DIR"

if [ "$SIGNED" = 1 ]; then
  echo "==> Signing the RPM packages (key ${GPG_FINGERPRINT})"
  if ! command -v rpmsign >/dev/null 2>&1; then
    echo "missing required tool for signing: rpmsign (apt install rpm)" >&2
    exit 1
  fi
  GPG_BIN=$(command -v gpg || command -v gpg2)
  for rpm_file in "$RPM_DIR"/*.rpm; do
    rpmsign --define "__gpg ${GPG_BIN}" --define "_gpg_name ${GPG_FINGERPRINT}" --addsign "$rpm_file" >/dev/null
  done
  rpm -q --qf '%{NAME} %{VERSION} %{SIGPGP:pgpsig}\n' "$RPM_DIR"/*.rpm
fi

echo "==> Assembling the APT repository"
APT_DIR="$SITE_DIR/apt"
POOL_DIR="$APT_DIR/pool/main/n/nido"
mkdir -p "$POOL_DIR"
cp "$DEB_DIR"/*.deb "$POOL_DIR/"

echo "==> Assembling the DNF repository"
RPM_REPO="$SITE_DIR/rpm"
mkdir -p "$RPM_REPO/Packages"
cp "$RPM_DIR"/*.rpm "$RPM_REPO/Packages/"

if [ -n "$PREVIOUS_SITE_DIR" ] && [ -d "$PREVIOUS_SITE_DIR" ]; then
  echo "==> Carrying over packages from the previous gh-pages tree"
  if [ -d "$PREVIOUS_SITE_DIR/apt/pool/main/n/nido" ]; then
    cp -n "$PREVIOUS_SITE_DIR"/apt/pool/main/n/nido/*.deb "$POOL_DIR/" 2>/dev/null || true
  fi
  if [ -d "$PREVIOUS_SITE_DIR/rpm/Packages" ]; then
    cp -n "$PREVIOUS_SITE_DIR"/rpm/Packages/*.rpm "$RPM_REPO/Packages/" 2>/dev/null || true
  fi
fi

python3 - "$POOL_DIR" "$RPM_REPO/Packages" "$KEEP_VERSIONS" <<'PYPRUNE'
import re
import sys
from pathlib import Path

pool, rpm_dir, keep = Path(sys.argv[1]), Path(sys.argv[2]), int(sys.argv[3])


def version_key(path):
    match = re.search(r"_([0-9][0-9A-Za-z.+~:-]*?)_[a-z0-9]+\.deb$", path.name)
    if match is None:
        match = re.search(r"-([0-9][0-9A-Za-z.+~]*?)-[0-9][^-]*\.[a-z0-9]+\.rpm$", path.name)
    if match is None:
        return ((0, 0),)
    parts = re.split(r"[.~\-]", match.group(1))
    return tuple((0, int(part)) if part.isdigit() else (1, part) for part in parts)


def prune(directory, pattern):
    versions = {}
    for path in directory.glob(pattern):
        versions.setdefault(version_key(path), []).append(path)
    for key in sorted(versions)[:-keep]:
        for path in versions[key]:
            path.unlink()
    print(f"pruned {directory}: kept {len(sorted(versions)[-keep:])} version(s)")


prune(pool, "*.deb")
prune(rpm_dir, "*.rpm")
PYPRUNE

# apt-ftparchive indexes the whole pool; split the stanzas per architecture so
# each binary-<arch>/Packages only advertises the packages a client can install.
( cd "$APT_DIR" && apt-ftparchive packages pool ) > "$WORK_DIR/Packages-all"
for arch in amd64 arm64; do
  binary_dir="$APT_DIR/dists/stable/main/binary-$arch"
  mkdir -p "$binary_dir"
  python3 - "$arch" "$WORK_DIR/Packages-all" > "$binary_dir/Packages" <<'PYSPLIT'
import sys

arch, source = sys.argv[1], sys.argv[2]
with open(source, encoding="utf-8", errors="replace") as handle:
    stanzas = handle.read().strip().split("\n\n")
selected = [s for s in stanzas if "\nArchitecture: " + arch + "\n" in ("\n" + s + "\n")]
sys.stdout.write("\n\n".join(selected) + ("\n" if selected else ""))
PYSPLIT
  gzip -9 -c "$binary_dir/Packages" > "$binary_dir/Packages.gz"
done
( cd "$APT_DIR" && apt-ftparchive \
    -o APT::FTPArchive::Release::Origin="Nido" \
    -o APT::FTPArchive::Release::Label="Nido" \
    -o APT::FTPArchive::Release::Suite="stable" \
    -o APT::FTPArchive::Release::Codename="stable" \
    -o APT::FTPArchive::Release::Architectures="amd64 arm64" \
    -o APT::FTPArchive::Release::Components="main" \
    release dists/stable ) > "$APT_DIR/dists/stable/Release"

if [ "$SIGNED" = 1 ]; then
  echo "==> Signing the APT Release"
  gpg --batch --yes --armor --detach-sign -o "$APT_DIR/dists/stable/Release.gpg" "$APT_DIR/dists/stable/Release"
  gpg --batch --yes --clearsign -o "$APT_DIR/dists/stable/InRelease" "$APT_DIR/dists/stable/Release"
fi

createrepo_c --database --update "$RPM_REPO"

if [ "$SIGNED" = 1 ]; then
  echo "==> Signing the DNF repomd"
  gpg --batch --yes --armor --detach-sign -o "$RPM_REPO/repodata/repomd.xml.asc" "$RPM_REPO/repodata/repomd.xml"
fi

if [ "$SIGNED" = 1 ]; then
  cat > "$RPM_REPO/nido.repo" <<REPO
[nido]
name=Nido
baseurl=https://josepavese.github.io/nido/rpm/
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://josepavese.github.io/nido/nido-archive-keyring.asc
REPO
else
  cat > "$RPM_REPO/nido.repo" <<REPO
[nido]
name=Nido
baseurl=https://josepavese.github.io/nido/rpm/
enabled=1
gpgcheck=0
REPO
fi

if [ "$SIGNED" = 1 ]; then
  APT_INSTRUCTIONS='sudo install -d -m 0755 /etc/apt/keyrings
sudo curl -fsSL https://josepavese.github.io/nido/nido-archive-keyring.asc -o /etc/apt/keyrings/nido.asc
echo "deb [signed-by=/etc/apt/keyrings/nido.asc] https://josepavese.github.io/nido/apt/ stable main" | sudo tee /etc/apt/sources.list.d/nido.list'
  FOOTER='The repositories are signed with the Nido package archive key
(fingerprint DBB87CB63A94ABA50B33C7B919C6A95C9D280B64, expires 2029-10-24);
the public keyring is served as <code>/nido-archive-keyring.asc</code>.'
else
  APT_INSTRUCTIONS='echo "deb [trusted=yes] https://josepavese.github.io/nido/apt/ stable main" | sudo tee /etc/apt/sources.list.d/nido.list'
  FOOTER='The repository is unsigned: APT uses <code>[trusted=yes]</code> and DNF
<code>gpgcheck=0</code>.'
fi

cat > "$SITE_DIR/index.html" <<HTML
<!doctype html>
<html lang="en">
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Nido package repositories</title>
<style>
  body { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; max-width: 52rem; margin: 3rem auto; padding: 0 1rem; line-height: 1.5; color: #1b1b1b; }
  pre { background: #f4f4f4; padding: 0.75rem 1rem; overflow-x: auto; border-radius: 6px; }
  h1 { font-size: 1.4rem; } h2 { font-size: 1.05rem; margin-top: 2rem; }
  code { background: #f4f4f4; padding: 0.1rem 0.25rem; border-radius: 3px; }
</style>
<h1>Nido $VERSION — APT and DNF repositories</h1>
<p>Packages for the <a href="https://github.com/Josepavese/nido">Nido local VM manager</a>.
QEMU is a host prerequisite; run <code>nido doctor</code> after installing.</p>
<h2>Debian / Ubuntu (APT)</h2>
<pre>$APT_INSTRUCTIONS
sudo apt-get update
sudo apt-get install nido</pre>
<h2>Fedora / RHEL (DNF)</h2>
<pre>sudo curl -fsSL https://josepavese.github.io/nido/rpm/nido.repo -o /etc/yum.repos.d/nido.repo
sudo dnf install nido</pre>
<p>$FOOTER</p>
</html>
HTML
: > "$SITE_DIR/.nojekyll"

echo "==> Site ready: $SITE_DIR (signed=$SIGNED)"
find "$SITE_DIR" -maxdepth 4 -type f | sort | sed "s|$SITE_DIR/||"
