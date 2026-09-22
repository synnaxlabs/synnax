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

# Stands in for gh: serves canned JSON from FAKE_GH_DIR for the two endpoints the script
# calls and applies the requested jq expression.
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
    */actions/workflows/*/runs)
        NAME=${ENDPOINT#*/actions/workflows/}
        FILE="$FAKE_GH_DIR/runs/${NAME%/runs}.json"
        ;;
    *)
        echo "unexpected endpoint: $ENDPOINT" >&2
        exit 1
        ;;
esac
jq -r "$EXPR" "$FILE"
"""

RUNS_URL = "https://github.com/synnaxlabs/synnax/actions/runs"


class Harness:
    """A checkout with workflow files and a fake gh serving check and run data."""

    def __init__(self, path: Path) -> None:
        self.root = path
        self.data = path / "gh"
        (self.data / "runs").mkdir(parents=True)
        (self.root / ".github" / "workflows").mkdir(parents=True)
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

    def workflow(
        self,
        name: str,
        trigger: str = "push",
        *,
        status: str | None = "completed",
        conclusion: str | None = "success",
    ) -> None:
        """Adds a workflow file; status None means it never ran on the branch."""
        (self.root / ".github" / "workflows" / name).write_text(
            f"on:\n  {trigger}:\n    branches:\n      - main\n"
        )
        runs = [] if status is None else [{"status": status, "conclusion": conclusion}]
        (self.data / "runs" / f"{name}.json").write_text(
            json.dumps({"workflow_runs": runs})
        )

    def run(self, run_id: int = 99) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT), "abc123", str(run_id), "main"],
            cwd=self.root,
            env=self.env,
            capture_output=True,
            text=True,
        )


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    harness = Harness(tmp_path)
    harness.check_runs(("Test", "completed", "success", 1))
    harness.workflow("test.core.yaml")
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


class TestBranchRuns:
    """Tests for the newest push run of each workflow on the branch."""

    def test_fails_when_the_newest_run_failed(self, harness: Harness) -> None:
        harness.workflow("test.pluto.yaml", conclusion="failure")
        result = harness.run()
        assert result.returncode != 0
        assert "test.pluto.yaml (failure)" in result.stderr
        assert "test.core.yaml" not in result.stderr

    def test_fails_when_the_newest_run_is_pending(self, harness: Harness) -> None:
        harness.workflow("lint.integration.yaml", status="in_progress", conclusion=None)
        result = harness.run()
        assert result.returncode != 0
        assert "lint.integration.yaml (in_progress)" in result.stderr

    def test_skips_a_workflow_that_never_ran(self, harness: Harness) -> None:
        harness.workflow("check.oracle.yaml", status=None)
        result = harness.run()
        assert result.returncode == 0, result.stderr
        assert "check.oracle.yaml never ran on main, skipped" in result.stdout
        assert "1 workflows pass on main" in result.stdout

    def test_accepts_a_skipped_run(self, harness: Harness) -> None:
        harness.workflow("test.docs.yaml", conclusion="skipped")
        assert harness.run().returncode == 0

    def test_ignores_workflows_without_a_push_trigger(self, harness: Harness) -> None:
        harness.workflow("test.go.yaml", "workflow_call", status=None)
        result = harness.run()
        assert result.returncode == 0, result.stderr
        assert "1 workflows pass on main" in result.stdout

    def test_ignores_deploy_workflows(self, harness: Harness) -> None:
        harness.workflow("deploy.ts.yaml", conclusion="failure")
        assert harness.run().returncode == 0
