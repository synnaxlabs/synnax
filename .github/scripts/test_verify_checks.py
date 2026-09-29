#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import subprocess
from pathlib import Path

import pytest

SCRIPT = Path(__file__).resolve().parent / "verify_checks.sh"

# Stands in for gh: serves the canned check runs from FAKE_GH_DIR and applies the
# requested jq expression.
FAKE_GH = """#!/usr/bin/env bash
set -euo pipefail
ENDPOINT=""
EXPR="."
while [ $# -gt 0 ]; do
    case "$1" in
        api | --paginate | --method | GET) shift ;;
        -f | -F) shift 2 ;;
        --jq)
            EXPR=$2
            shift 2
            ;;
        *)
            ENDPOINT=$1
            shift
            ;;
    esac
done
case "$ENDPOINT" in
    */check-runs*) FILE="$FAKE_GH_DIR/check-runs.json" ;;
    *)
        echo "unexpected endpoint: $ENDPOINT" >&2
        exit 1
        ;;
esac
jq -r "$EXPR" "$FILE"
"""

RUNS_URL = "https://github.com/synnaxlabs/synnax/actions/runs"


class Harness:
    """A fake gh serving check run data."""

    def __init__(self, path: Path) -> None:
        self.root = path
        self.data = path / "gh"
        self.data.mkdir()
        bin_dir = path / "bin"
        bin_dir.mkdir()
        gh = bin_dir / "gh"
        gh.write_text(FAKE_GH)
        gh.chmod(0o755)
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "FAKE_GH_DIR": str(self.data),
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }
        self.check_runs()

    def check_runs(self, *runs: tuple[str, str, str | None, int]) -> None:
        """Sets the check runs on the commit as (name, status, conclusion, run id)."""
        payload = {
            "check_runs": [
                {
                    "name": name,
                    "status": status,
                    "conclusion": conclusion,
                    "app": {"slug": "github-actions"},
                    "html_url": f"{RUNS_URL}/{run_id}/job/1",
                }
                for name, status, conclusion, run_id in runs
            ]
        }
        (self.data / "check-runs.json").write_text(json.dumps(payload))

    def run(self, run_id: int = 99) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT), "abc123", str(run_id)],
            cwd=self.root,
            env=self.env,
            capture_output=True,
            text=True,
        )


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    harness = Harness(tmp_path)
    harness.check_runs(("Test", "completed", "success", 1))
    return harness


class TestCheckRuns:
    """Tests for the check runs on the commit."""

    def test_passes(self, harness: Harness) -> None:
        result = harness.run()
        assert result.returncode == 0, result.stderr
        assert "1 checks passed on abc123" in result.stdout

    def test_ignores_the_current_run(self, harness: Harness) -> None:
        harness.check_runs(
            ("Test", "completed", "success", 1),
            ("Release", "in_progress", None, 99),
        )
        assert harness.run(run_id=99).returncode == 0

    def test_fails_with_no_check_runs(self, harness: Harness) -> None:
        harness.check_runs()
        result = harness.run()
        assert result.returncode != 0
        assert "no check runs found" in result.stderr

    def test_fails_on_a_pending_check(self, harness: Harness) -> None:
        harness.check_runs(("Test", "queued", None, 1))
        result = harness.run()
        assert result.returncode != 0
        assert "Test (queued)" in result.stderr

    def test_fails_on_a_failed_check(self, harness: Harness) -> None:
        harness.check_runs(("Test", "completed", "failure", 1))
        result = harness.run()
        assert result.returncode != 0
        assert "Test (failure)" in result.stderr

    def test_accepts_skipped_checks(self, harness: Harness) -> None:
        harness.check_runs(("Test", "completed", "skipped", 1))
        assert harness.run().returncode == 0

