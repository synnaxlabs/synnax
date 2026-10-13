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

SCRIPT = Path(__file__).resolve().parent / "check_review.sh"

# Stands in for gh: serves canned JSON from FAKE_GH_DIR for the endpoints the script
# calls and applies the requested jq expression. A commit with no canned check runs
# serves an empty list.
FAKE_GH = """#!/usr/bin/env bash
set -euo pipefail
ENDPOINT=""
EXPR="."
while [ $# -gt 0 ]; do
    case "$1" in
        api | --paginate) shift ;;
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
    graphql) FILE="$FAKE_GH_DIR/edits.json" ;;
    */commits/*/check-runs*)
        SHA=${ENDPOINT#*/commits/}
        FILE="$FAKE_GH_DIR/check-runs-${SHA%%/*}.json"
        [ -f "$FILE" ] || FILE="$FAKE_GH_DIR/check-runs-none.json"
        ;;
    */pulls/*) FILE="$FAKE_GH_DIR/pull.json" ;;
    *)
        echo "unexpected endpoint: $ENDPOINT" >&2
        exit 1
        ;;
esac
jq -r "$EXPR" "$FILE"
"""


class Harness:
    """A fake gh serving one pull request's head, its check runs, and its edits."""

    def __init__(self, path: Path) -> None:
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
        (self.data / "check-runs-none.json").write_text('{"check_runs": []}')
        (self.data / "pull.json").write_text(json.dumps({"head": {"sha": "head"}}))
        self.check_runs("head", ("Greptile Review", "success"))
        self.edits(greptile_edit(5))

    def check_runs(self, sha: str, *runs: tuple[str, str | None]) -> None:
        """Sets the (name, conclusion) check runs a commit reports."""
        payload = {"check_runs": [{"name": n, "conclusion": c} for n, c in runs]}
        (self.data / f"check-runs-{sha}.json").write_text(json.dumps(payload))

    def edits(self, *edits: dict) -> None:
        """Sets the description revisions, newest first, as GitHub serves them."""
        payload = {
            "data": {
                "repository": {
                    "pullRequest": {"userContentEdits": {"nodes": list(edits)}}
                }
            }
        }
        (self.data / "edits.json").write_text(json.dumps(payload))

    def run(self) -> tuple[str, str]:
        """Runs the script and returns the (state, description) it reports."""
        result = subprocess.run(
            [str(SCRIPT), "42"], env=self.env, capture_output=True, text=True
        )
        assert result.returncode == 0, result.stderr
        state, description = result.stdout.rstrip("\n").split("\t")
        return state, description


def edit(body: str | None, login: str = "author", kind: str = "User") -> dict:
    """A description revision by an account of the given kind."""
    return {"editor": {"__typename": kind, "login": login}, "diff": body}


def greptile_body(*scores: int, sha: str = "head") -> str:
    """A description the way Greptile leaves it after it scores a commit."""
    markers = "".join(f"<!-- greptile_confidence_score:{s} -->\n" for s in scores)
    link = f"https://github.com/synnaxlabs/synnax/commit/{sha})"
    return f'{markers}<sub>Last reviewed commit: ["Fix"]({link}</sub>'


def greptile_edit(*scores: int, sha: str = "head") -> dict:
    """A description revision Greptile wrote after it scores a commit."""
    return edit(greptile_body(*scores, sha=sha), "greptile-apps", "Bot")


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestReview:
    """Tests for the Greptile check run requirement."""

    def test_passes_a_reviewed_head(self, harness: Harness) -> None:
        assert harness.run() == ("success", "Greptile scored the head 5/5")

    def test_pends_without_a_review(self, harness: Harness) -> None:
        harness.check_runs("head")
        state, description = harness.run()
        assert state == "pending"
        assert description == "waiting for the Greptile review of the head"

    def test_pends_while_the_review_runs(self, harness: Harness) -> None:
        harness.check_runs("head", ("Greptile Review", None))
        assert harness.run()[0] == "pending"

    def test_ignores_another_check_run(self, harness: Harness) -> None:
        harness.check_runs("head", ("Check Formatting", "success"))
        assert harness.run()[0] == "pending"

    def test_rejects_a_review_of_an_earlier_commit(self, harness: Harness) -> None:
        harness.check_runs("head")
        harness.check_runs("earlier", ("Greptile Review", "success"))
        assert harness.run()[0] == "pending"


class TestScore:
    """Tests for the Greptile confidence score requirement."""

    def test_fails_below_five(self, harness: Harness) -> None:
        harness.edits(greptile_edit(4))
        state, description = harness.run()
        assert state == "failure"
        assert description == "Greptile scored the head 4/5, not 5/5"

    def test_pends_without_a_score(self, harness: Harness) -> None:
        harness.edits(edit("A description"))
        state, description = harness.run()
        assert state == "pending"
        assert description == "waiting for Greptile to score the head"

    def test_rejects_a_score_of_an_earlier_commit(self, harness: Harness) -> None:
        harness.edits(greptile_edit(5, sha="earlier"))
        assert harness.run()[0] == "pending"

    def test_ignores_a_score_the_author_wrote(self, harness: Harness) -> None:
        harness.edits(edit(greptile_body(5)), greptile_edit(2))
        assert harness.run()[0] == "failure"

    def test_ignores_a_user_with_the_login_of_greptile(self, harness: Harness) -> None:
        harness.edits(edit(greptile_body(5), "greptile-apps"))
        assert harness.run()[0] == "pending"

    def test_takes_the_latest_score(self, harness: Harness) -> None:
        harness.edits(greptile_edit(5), greptile_edit(2, sha="earlier"))
        assert harness.run()[0] == "success"

    def test_rejects_a_second_marker_that_is_not_a_five(self, harness: Harness) -> None:
        harness.edits(greptile_edit(3, 5))
        assert harness.run()[0] == "failure"

    def test_pends_on_a_deleted_revision(self, harness: Harness) -> None:
        harness.edits(edit(None, "greptile-apps", "Bot"), greptile_edit(5))
        assert harness.run()[0] == "pending"
