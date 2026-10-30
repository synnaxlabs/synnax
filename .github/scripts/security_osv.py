#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Checks the C and C++ dependencies that Dependabot cannot read against OSV, and
prints the vulnerabilities as SARIF.

Usage: security_osv.py, from the repository root.

Checks each Git submodule by its commit, and each Bazel module in BAZEL by its release
tag. OSV has no Git data for the other Bazel modules.
"""

import json
import re
import subprocess
import sys
import urllib.request
from pathlib import Path
from typing import Any

OSV_URL = "https://api.osv.dev/v1/query"
# Bazel module name to its upstream repository and its release tag for a version.
BAZEL = {
    "cpp-httplib": ("https://github.com/yhirose/cpp-httplib.git", "v{}"),
    "curl": ("https://github.com/curl/curl.git", "curl-{}"),
    "grpc": ("https://github.com/grpc/grpc.git", "v{}"),
    "openssl": ("https://github.com/openssl/openssl.git", "openssl-{}"),
}
BAZEL_DEP = re.compile(
    r'bazel_dep\(\s*name\s*=\s*"([^"]+)"\s*,\s*version\s*=\s*"([^"]+)"'
)


def tag(name: str, version: str) -> str:
    """Returns the upstream release tag of a Bazel module version."""
    version = re.sub(r"\.bcr\.\d+$", "", version)
    if name == "curl":
        version = version.replace(".", "_")
    return BAZEL[name][1].format(version)


def queries(module: str, gitlinks: str) -> list[tuple[str, str, dict[str, Any]]]:
    """Returns (location, dependency, OSV query) for each dependency to check."""
    found = []
    versions = dict(BAZEL_DEP.findall(module))
    if missing := sorted(BAZEL.keys() - versions.keys()):
        raise ValueError(f"MODULE.bazel has no version for {', '.join(missing)}")
    for name in sorted(BAZEL):
        package = {"name": BAZEL[name][0], "ecosystem": "GIT"}
        query = {"package": package, "version": tag(name, versions[name])}
        found.append(("MODULE.bazel", f"{name} {versions[name]}", query))
    for line in gitlinks.splitlines():
        mode, commit, _, path = line.split(maxsplit=3)
        if mode == "160000":
            found.append((path, f"{path} {commit[:12]}", {"commit": commit}))
    return found


def query_osv(query: dict[str, Any], url: str = OSV_URL) -> list[dict[str, Any]]:
    request = urllib.request.Request(
        url,
        data=json.dumps(query).encode(),
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=60) as response:
        vulns: list[dict[str, Any]] = json.load(response).get("vulns", [])
    return vulns


def sarif(found: list[tuple[str, str, list[dict[str, Any]]]]) -> dict[str, Any]:
    results = [
        {
            "ruleId": vuln["id"],
            "level": "error",
            "message": {"text": f"{dependency}: {vuln.get('summary', '')}"},
            "locations": [{"physicalLocation": {"artifactLocation": {"uri": path}}}],
        }
        for path, dependency, vulns in found
        for vuln in vulns
    ]
    return {
        "version": "2.1.0",
        "runs": [{"tool": {"driver": {"name": "osv"}}, "results": results}],
    }


def main(url: str = OSV_URL) -> int:
    module = Path("MODULE.bazel").read_text()
    gitlinks = subprocess.check_output(["git", "ls-files", "-s"], text=True)
    found = [(p, d, query_osv(q, url)) for p, d, q in queries(module, gitlinks)]
    print(json.dumps(sarif(found)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
