#!/usr/bin/env python3
"""Retain package versions together across all published host architectures."""
import re
import sys
from pathlib import Path

def version_key(path):
    match = re.search(r"_([0-9][0-9A-Za-z.+~:-]*?)_[a-z0-9]+\.deb$", path.name)
    if match is None:
        match = re.search(r"-([0-9][0-9A-Za-z.+~]*?)-[0-9][^-]*\.[a-z0-9_]+\.rpm$", path.name)
    if match is None:
        raise ValueError(f"unrecognized package filename: {path.name}")
    parts = re.split(r"[.~\-]", match.group(1))
    return tuple((0, int(part)) if part.isdigit() else (1, part) for part in parts)


def prune(directory, pattern, keep):
    versions = {}
    for path in directory.glob(pattern):
        versions.setdefault(version_key(path), []).append(path)
    for key in sorted(versions)[:-keep]:
        for path in versions[key]:
            path.unlink()
    print(f"pruned {directory}: kept {len(sorted(versions)[-keep:])} version(s)")


if __name__ == "__main__":
    pool, rpm_dir, keep = Path(sys.argv[1]), Path(sys.argv[2]), int(sys.argv[3])
    if keep <= 0:
        raise ValueError("retained version count must be positive")
    prune(pool, "*.deb", keep)
    prune(rpm_dir, "*.rpm", keep)
