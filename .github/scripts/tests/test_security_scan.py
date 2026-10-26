#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import subprocess
import sys
from pathlib import Path

import pytest

SCRIPT = Path(__file__).parents[1] / "security_scan.sh"
SARIF = '{"runs": []}'
FAKES = {
    # Fails in module b/c, so the test sees its exit code.
    "govulncheck": (
        f'echo stderr >&2; [ "$(basename "$PWD")" = c ] && exit 3; echo \'{SARIF}\''
    ),
    "gitleaks": f"echo '{SARIF}'",
    "zizmor": f"echo '{SARIF}'",
    "checkov": (
        'while [ $# -gt 0 ]; do [ "$1" = --output-file-path ] && dir=$2; shift; done; '
        f'mkdir -p "$dir"; echo \'{SARIF}\' > "$dir/results_sarif.sarif"; echo banner'
    ),
}


@pytest.fixture
def repo(tmp_path: Path) -> Path:
    root = tmp_path / "repo"
    for path in ("a/go.mod", "b/c/go.mod", "deploy/preview/main.tf"):
        (root / path).parent.mkdir(parents=True, exist_ok=True)
        (root / path).write_text("")
    subprocess.run(["git", "init", "-q"], cwd=root, check=True)
    subprocess.run(["git", "add", "."], cwd=root, check=True)
    return root


@pytest.mark.skipif(sys.platform == "win32", reason="runs a POSIX shell script")
def test_should_write_sarif_log_and_exit_code_per_scan(
    repo: Path, tmp_path: Path
) -> None:
    """Should run each scanner once per target and keep its output out of stdout."""
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for name, body in FAKES.items():
        (bin_dir / name).write_text(f"#!/bin/sh\n{body}\n")
        (bin_dir / name).chmod(0o755)
    out = tmp_path / "out"
    env = {**os.environ, "PATH": f"{bin_dir}:{os.environ['PATH']}"}
    done = subprocess.run(
        ["bash", str(SCRIPT), str(out)], cwd=repo, env=env, capture_output=True
    )
    assert (done.returncode, done.stdout, done.stderr) == (0, b"", b"")
    exits = {p.stem: p.read_text().strip() for p in out.glob("*.exit")}
    assert exits == {
        "govulncheck-a": "0",
        "govulncheck-b-c": "3",
        "gitleaks": "0",
        "zizmor": "0",
        "checkov": "0",
    }
    assert (out / "govulncheck-a.sarif").read_text().strip() == SARIF
    assert (out / "govulncheck-a.log").read_text() == "stderr\n"
    assert (out / "checkov.sarif").read_text().strip() == SARIF
    assert (out / "checkov.log").read_text() == "banner\n"
