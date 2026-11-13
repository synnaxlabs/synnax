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
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "runner_core_source.sh"

RUNS = "https://github.com/synnaxlabs/synnax/actions/runs"
LINK = "https://artifacts.example.com/core.zip?sig=abc%3D&se=1"

# Prints the artifact ids that the test gives, as the jq filter of the script would.
GH = """#!/usr/bin/env bash
echo "gh $1 $2 $3" >> "$LOG"
cat "$IDS"
"""

# Prints the redirect that the test gives, as curl does for its --write-out.
CURL = """#!/usr/bin/env bash
echo "curl ${@: -1}" >> "$LOG"
printf '%s' "$REDIRECT"
"""


def source(
    tmp_path: Path, core: str, ids: tuple[int, ...] = (), redirect: str = LINK
) -> tuple[subprocess.CompletedProcess[str], list[str]]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for name, body in (("gh", GH), ("curl", CURL)):
        tool = bin_dir / name
        tool.write_text(body)
        tool.chmod(0o755)
    ids_file = tmp_path / "ids"
    ids_file.write_text("".join(f"{i}\n" for i in ids))
    log = tmp_path / "log"
    result = subprocess.run(
        [str(SCRIPT), core],
        env={
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "LOG": str(log),
            "IDS": str(ids_file),
            "REDIRECT": redirect,
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
            "GH_TOKEN": "token",
        },
        capture_output=True,
        text=True,
    )
    calls = log.read_text().splitlines() if log.exists() else []
    return result, calls


class TestRunnerCoreSource:
    """Tests for finding the Core binary that a release version or a run URL names."""

    @pytest.mark.parametrize("version", ["0.61.0", "0.62.0-rc.1"])
    def test_should_give_the_binary_of_a_release(
        self, tmp_path: Path, version: str
    ) -> None:
        result, calls = source(tmp_path, version)
        assert result.returncode == 0
        assert result.stdout.splitlines() == [
            "$Url = 'https://github.com/synnaxlabs/synnax/releases/download/"
            f"core/v{version}/synnax-v{version}-windows.exe'",
            f"$Version = '{version}'",
        ]
        assert calls == []

    @pytest.mark.parametrize("url", [f"{RUNS}/123", f"{RUNS}/123/job/456?pr=7"])
    def test_should_give_the_artifact_of_a_run_or_a_job(
        self, tmp_path: Path, url: str
    ) -> None:
        result, calls = source(tmp_path, url, ids=(9,))
        assert result.returncode == 0
        assert result.stdout.splitlines() == [f"$Url = '{LINK}'", "$Version = ''"]
        assert calls == [
            "gh api --paginate repos/synnaxlabs/synnax/actions/runs/123/artifacts",
            "curl https://api.github.com/repos/synnaxlabs/synnax/actions/artifacts/9"
            "/zip",
        ]

    @pytest.mark.parametrize("ids", [(), (9, 10)])
    def test_should_fail_unless_the_run_has_one_artifact(
        self, tmp_path: Path, ids: tuple[int, ...]
    ) -> None:
        result, _ = source(tmp_path, f"{RUNS}/123", ids=ids)
        assert result.returncode == 1
        assert f"Run 123 has {len(ids)} Windows Core artifacts" in result.stderr
        assert result.stdout == ""

    def test_should_fail_without_a_download_link(self, tmp_path: Path) -> None:
        result, _ = source(tmp_path, f"{RUNS}/123", ids=(9,), redirect="")
        assert result.returncode == 1
        assert "no download link for artifact 9" in result.stderr
        assert result.stdout == ""

    @pytest.mark.parametrize("core", ["latest", "v0.61.0", "123", "0.61"])
    def test_should_fail_on_any_other_input(self, tmp_path: Path, core: str) -> None:
        result, calls = source(tmp_path, core)
        assert result.returncode == 1
        assert "not a release version" in result.stderr
        assert calls == []
