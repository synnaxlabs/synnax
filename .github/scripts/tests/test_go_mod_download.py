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

SCRIPT = Path(__file__).resolve().parents[1] / "go_mod_download.sh"

# Fails while the directory's fails file holds a count above zero, and counts it down.
GO = """#!/usr/bin/env bash
echo "$2" >> "$LOG"
fails=$(cat "$2/fails" 2>/dev/null || echo 0)
if [ "$fails" -gt 0 ]; then
    echo $((fails - 1)) > "$2/fails"
    exit 1
fi
"""

SLEEP = """#!/usr/bin/env bash
echo "sleep $1" >> "$LOG"
"""


def download(
    tmp_path: Path, fails: dict[str, int]
) -> tuple[subprocess.CompletedProcess[str], list[str]]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for name, body in (("go", GO), ("sleep", SLEEP)):
        tool = bin_dir / name
        tool.write_text(body)
        tool.chmod(0o755)
    for directory, count in fails.items():
        (tmp_path / directory).mkdir()
        (tmp_path / directory / "fails").write_text(str(count))
    log = tmp_path / "log"
    result = subprocess.run(
        [str(SCRIPT), *fails],
        cwd=tmp_path,
        env={**os.environ, "PATH": f"{bin_dir}:{os.environ['PATH']}", "LOG": str(log)},
        capture_output=True,
        text=True,
    )
    calls = log.read_text().splitlines() if log.exists() else []
    return result, calls


class TestGoModDownload:
    """Tests for downloading Go modules with retries."""

    def test_should_download_each_directory_once(self, tmp_path: Path) -> None:
        result, calls = download(tmp_path, {"a": 0, "b": 0})
        assert result.returncode == 0
        assert calls == ["a", "b"]

    def test_should_retry_with_a_growing_wait(self, tmp_path: Path) -> None:
        result, calls = download(tmp_path, {"a": 2})
        assert result.returncode == 0
        assert calls == ["a", "sleep 15", "a", "sleep 30", "a"]

    def test_should_fail_after_three_attempts(self, tmp_path: Path) -> None:
        result, calls = download(tmp_path, {"a": 3, "b": 0})
        assert result.returncode == 1
        assert calls == ["a", "sleep 15", "a", "sleep 30", "a"]
        assert "::error::go mod download failed 3 times in a" in result.stdout

    def test_should_reset_the_attempts_for_each_directory(self, tmp_path: Path) -> None:
        result, calls = download(tmp_path, {"a": 2, "b": 2})
        assert result.returncode == 0
        assert calls == [
            "a",
            "sleep 15",
            "a",
            "sleep 30",
            "a",
            "b",
            "sleep 15",
            "b",
            "sleep 30",
            "b",
        ]

    def test_should_require_a_directory(self, tmp_path: Path) -> None:
        result, calls = download(tmp_path, {})
        assert result.returncode == 2
        assert calls == []
