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

SCRIPT = Path(__file__).resolve().parents[1] / "download_run_artifacts.sh"

EGRESS = "error downloading core: HTTP 503: 503 Egress is over the account limit."
MISSING = "no artifact matches any of the names or patterns provided"

# Fails with each line of the errors file in turn, then downloads. A line that starts
# with "always: " repeats forever. A failure leaves a stale file, as a partial download
# does, and gh refuses to extract over one.
GH = """#!/usr/bin/env bash
echo "gh $*" >> "$LOG"
if [ -e "$DIR/stale" ]; then
    echo "error extracting: file exists" >&2
    exit 1
fi
mkdir -p "$DIR"
line=$(head -n 1 "$ERRORS")
case "$line" in
    always:*) echo "${line#always: }" >&2; exit 1 ;;
esac
if [ -n "$line" ]; then
    tail -n +2 "$ERRORS" > "$ERRORS.next"
    mv "$ERRORS.next" "$ERRORS"
    touch "$DIR/stale"
    echo "$line" >&2
    exit 1
fi
touch "$DIR/binary"
"""

SLEEP = """#!/usr/bin/env bash
echo "sleep $1" >> "$LOG"
"""


def download(
    tmp_path: Path, errors: list[str]
) -> tuple[subprocess.CompletedProcess[str], list[str]]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    for name, body in (("gh", GH), ("sleep", SLEEP)):
        tool = bin_dir / name
        tool.write_text(body)
        tool.chmod(0o755)
    errors_file = tmp_path / "errors"
    errors_file.write_text("".join(f"{error}\n" for error in errors))
    log = tmp_path / "log"
    directory = tmp_path / "binaries"
    result = subprocess.run(
        [str(SCRIPT), "123", "synnax-core*linux", str(directory)],
        env={
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "LOG": str(log),
            "ERRORS": str(errors_file),
            "DIR": str(directory),
        },
        capture_output=True,
        text=True,
    )
    calls = log.read_text().splitlines() if log.exists() else []
    return result, calls


def sleeps(calls: list[str]) -> list[int]:
    return [int(call.split()[1]) for call in calls if call.startswith("sleep")]


class TestDownloadRunArtifacts:
    """Tests for downloading run artifacts with retries on transient errors."""

    def test_should_download_once_on_success(self, tmp_path: Path) -> None:
        result, calls = download(tmp_path, [])
        assert result.returncode == 0
        assert calls == [
            f"gh run download 123 -p synnax-core*linux --dir {tmp_path}/binaries"
        ]
        assert (tmp_path / "binaries" / "binary").exists()

    def test_should_retry_an_egress_throttle_from_a_clean_directory(
        self, tmp_path: Path
    ) -> None:
        result, calls = download(tmp_path, [EGRESS, EGRESS])
        assert result.returncode == 0
        assert len(sleeps(calls)) == 2
        assert not (tmp_path / "binaries" / "stale").exists()
        assert "::warning::" in result.stdout

    def test_should_retry_network_errors(self, tmp_path: Path) -> None:
        errors = ["read: connection reset by peer", "net/http: TLS handshake timeout"]
        result, calls = download(tmp_path, errors)
        assert result.returncode == 0
        assert len(sleeps(calls)) == 2

    def test_should_fail_at_once_on_a_missing_artifact(self, tmp_path: Path) -> None:
        result, calls = download(tmp_path, [MISSING])
        assert result.returncode == 1
        assert sleeps(calls) == []
        assert MISSING in result.stdout
        assert "::error::" in result.stdout

    def test_should_fail_at_once_on_a_not_found_run(self, tmp_path: Path) -> None:
        error = "error fetching artifacts: HTTP 404: Not Found (https://api.github.com)"
        result, calls = download(tmp_path, [error])
        assert result.returncode == 1
        assert sleeps(calls) == []

    def test_should_back_off_and_stop_inside_the_retry_window(
        self, tmp_path: Path
    ) -> None:
        result, calls = download(tmp_path, [f"always: {EGRESS}"])
        assert result.returncode == 1
        waits = sleeps(calls)
        assert 15 <= waits[0] <= 22
        assert 30 <= waits[1] <= 45
        assert max(waits) <= 180
        assert sum(waits) <= 600
        assert sum(waits) >= 420
        assert "still failing" in result.stdout

    def test_should_require_three_arguments(self) -> None:
        result = subprocess.run([str(SCRIPT), "123"], capture_output=True, text=True)
        assert result.returncode == 2
