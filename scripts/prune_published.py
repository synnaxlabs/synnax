#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Removes distribution files from dist/ that already exist on PyPI.

PyPI file names are immutable: once a file is published it can never be uploaded
again, and rebuilding an unchanged package does not reproduce the published bytes
across time (a build backend upgrade alone rewrites the wheel's WHEEL file), so
uv's hash-based `--check-url` skipping fails for unchanged packages. Pruning by
published file name limits `uv publish` to genuinely new files. Run by the deploy
pipeline between `uv build` and `uv publish` (see SY-4519).

Usage: prune_published.py [dist-dir]
"""

import json
import re
import sys
import urllib.error
import urllib.request
from functools import cache
from pathlib import Path


def parse_name_version(filename: str) -> tuple[str, str]:
    """Extracts the PEP 503 normalized project name and version from a wheel or
    sdist file name."""
    if filename.endswith(".whl"):
        name, version = filename.split("-")[:2]
    elif filename.endswith(".tar.gz"):
        name, version = filename.removesuffix(".tar.gz").rsplit("-", 1)
    else:
        raise ValueError(f"unrecognized distribution file: {filename}")
    return re.sub(r"[-_.]+", "-", name).lower(), version


@cache
def published_files(name: str, version: str) -> frozenset[str]:
    """Returns the file names already published on PyPI for the given release, or
    an empty set if the release does not exist."""
    url = f"https://pypi.org/pypi/{name}/{version}/json"
    try:
        with urllib.request.urlopen(url) as resp:
            release = json.load(resp)
    except urllib.error.HTTPError as e:
        if e.code == 404:
            return frozenset()
        raise
    return frozenset(f["filename"] for f in release["urls"])


def main() -> None:
    dist_dir = (
        Path(sys.argv[1])
        if len(sys.argv) > 1
        else Path(__file__).resolve().parent.parent / "dist"
    )
    if not dist_dir.is_dir():
        sys.exit(f"❌ distribution directory not found: {dist_dir}")
    dists = [p for p in dist_dir.iterdir() if p.name.endswith((".whl", ".tar.gz"))]
    for file in sorted(dists):
        name, version = parse_name_version(file.name)
        if file.name in published_files(name, version):
            print(f"↷ {file.name} already on PyPI, pruning")
            file.unlink()
        else:
            print(f"→ {file.name} will be published")


if __name__ == "__main__":
    main()
