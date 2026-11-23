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
import time
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "review_mark.sh"


def run(
    tmp_path: Path, labels: list[str], fails: bool = False
) -> subprocess.CompletedProcess[str]:
    bin_dir = tmp_path / "bin"
    bin_dir.mkdir()
    gh = bin_dir / "gh"
    body = "exit 1" if fails else "printf '%s' \"$LABELS\""
    gh.write_text(f"#!/bin/sh\n{body}\n")
    gh.chmod(0o755)
    env = {
        **os.environ,
        "PATH": f"{bin_dir}{os.pathsep}{os.environ['PATH']}",
        "PR": "12",
        "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        "LABELS": "".join(f"{label}\n" for label in labels),
    }
    return subprocess.run([str(SCRIPT)], env=env, capture_output=True, text=True)


class TestReviewMark:
    """Tests for reading whether a pull request waits on a human review."""

    @pytest.mark.parametrize(
        ("labels", "mark"),
        [
            (["console", "review/human"], "human"),
            (["review/bot"], "none"),
            (["review/humane"], "none"),
            ([], "none"),
        ],
    )
    def test_should_print_the_mark_and_the_time(
        self, tmp_path: Path, labels: list[str], mark: str
    ) -> None:
        before = int(time.time())
        result = run(tmp_path, labels)
        assert result.returncode == 0, result.stderr
        printed, at = result.stdout.split()
        assert printed == mark
        assert before <= int(at) <= int(time.time())

    def test_should_fail_when_the_labels_cannot_be_read(self, tmp_path: Path) -> None:
        result = run(tmp_path, ["review/human"], fails=True)
        assert result.returncode != 0
        assert result.stdout == ""
