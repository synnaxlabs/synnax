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
    */pulls/*/reviews*) FILE="$FAKE_GH_DIR/reviews.json" ;;
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
    """A fake gh serving one pull request's labels, head checks, edits, and reviews."""

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
        self.pull()
        self.reviews()
        self.check_runs("head", ("Greptile Review", "success"))
        self.edits(greptile_edit(5))

    def pull(self, *labels: str, author: str = "author") -> None:
        payload = {
            "user": {"login": author},
            "head": {"sha": "head"},
            "labels": [{"name": l} for l in labels],
        }
        (self.data / "pull.json").write_text(json.dumps(payload))

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

    def reviews(self, *reviews: tuple[str, str] | tuple[str, str, str]) -> None:
        """Sets the reviews in order as (login, state) or (login, state, user type).

        The first review lands on its own page and the rest on a second one, the way gh
        --paginate emits them.
        """
        payload = [
            {
                "user": {"login": r[0], "type": r[2] if len(r) > 2 else "User"},
                "state": r[1],
            }
            for r in reviews
        ]
        pages = [payload[:1], payload[1:]] if len(payload) > 1 else [payload]
        (self.data / "reviews.json").write_text("\n".join(map(json.dumps, pages)))

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
    """A description revision Greptile wrote after it scored a commit."""
    return edit(greptile_body(*scores, sha=sha), "greptile-apps", "Bot")


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestLabels:
    """Tests for the tier label requirement."""

    def test_pends_without_a_tier_label(self, harness: Harness) -> None:
        harness.pull("console")
        state, description = harness.run()
        assert state == "pending"
        assert "add one review tier label" in description

    def test_fails_with_two_tier_labels(self, harness: Harness) -> None:
        harness.pull("review/light", "review/bot")
        state, description = harness.run()
        assert state == "failure"
        assert "found review/light review/bot" in description

    def test_ignores_other_labels(self, harness: Harness) -> None:
        harness.pull("console", "review/bot", "do not merge")
        state, description = harness.run()
        assert state == "success"
        assert "no human approval required" in description


class TestGreptile:
    """Tests for the Greptile review requirement."""

    def test_pends_without_a_review(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.check_runs("head")
        state, description = harness.run()
        assert state == "pending"
        assert description == "review/bot: waiting for the Greptile review of the head"

    def test_pends_while_the_review_runs(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.check_runs("head", ("Greptile Review", None))
        assert harness.run()[0] == "pending"

    def test_ignores_another_check_run(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.check_runs("head", ("Check Formatting", "success"))
        assert harness.run()[0] == "pending"

    def test_rejects_a_review_of_an_earlier_commit(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.check_runs("head")
        harness.check_runs("earlier", ("Greptile Review", "success"))
        assert harness.run()[0] == "pending"

    def test_fails_below_five(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(greptile_edit(4))
        state, description = harness.run()
        assert state == "failure"
        assert "Greptile scored the head 4/5" in description

    def test_fails_below_five_with_an_approval(self, harness: Harness) -> None:
        harness.pull("review/thorough")
        harness.edits(greptile_edit(3))
        harness.reviews(("reviewer", "APPROVED"))
        assert harness.run()[0] == "failure"

    def test_pends_without_a_score(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(edit("A description"))
        state, description = harness.run()
        assert state == "pending"
        assert description == "review/bot: waiting for Greptile to score the head"

    def test_rejects_a_score_of_an_earlier_commit(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(greptile_edit(5, sha="earlier"))
        assert harness.run()[0] == "pending"

    def test_ignores_a_score_the_author_wrote(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(edit(greptile_body(5)), greptile_edit(2))
        assert harness.run()[0] == "failure"

    def test_ignores_a_user_with_the_login_of_greptile(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(edit(greptile_body(5), "greptile-apps"))
        assert harness.run()[0] == "pending"

    def test_takes_the_latest_score(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(greptile_edit(5), greptile_edit(2, sha="earlier"))
        assert harness.run()[0] == "success"

    def test_rejects_a_second_marker_that_is_not_a_five(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(greptile_edit(3, 5))
        assert harness.run()[0] == "failure"

    def test_pends_on_a_deleted_revision(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.edits(edit(None, "greptile-apps", "Bot"), greptile_edit(5))
        assert harness.run()[0] == "pending"

    def test_a_missing_tier_label_outranks_it(self, harness: Harness) -> None:
        harness.pull()
        harness.check_runs("head")
        assert "add one review tier label" in harness.run()[1]


class TestApproval:
    """Tests for the human approval a tier needs."""

    def test_pends_without_an_approval(self, harness: Harness) -> None:
        harness.pull("review/light")
        state, description = harness.run()
        assert state == "pending"
        assert "waiting for an approval from someone but author" in description

    def test_passes_with_one_approval(self, harness: Harness) -> None:
        harness.pull("review/thorough")
        harness.reviews(("reviewer", "APPROVED"))
        assert harness.run() == ("success", "review/thorough: approved by reviewer")

    def test_ignores_the_author(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("author", "APPROVED"))
        assert harness.run()[0] == "pending"

    def test_ignores_bots(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("greptile-apps[bot]", "APPROVED", "Bot"))
        assert harness.run()[0] == "pending"

    def test_a_later_request_for_changes_retires_an_approval(
        self, harness: Harness
    ) -> None:
        harness.pull("review/light")
        harness.reviews(("reviewer", "APPROVED"), ("reviewer", "CHANGES_REQUESTED"))
        assert harness.run()[0] == "pending"

    def test_a_comment_does_not_retire_an_approval(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("reviewer", "APPROVED"), ("reviewer", "COMMENTED"))
        assert harness.run()[0] == "success"

    def test_a_dismissed_approval_does_not_count(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("reviewer", "DISMISSED"))
        assert harness.run()[0] == "pending"
