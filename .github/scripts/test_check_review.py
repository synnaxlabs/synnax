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

# Stands in for gh: serves canned JSON from FAKE_GH_DIR for the three pull request
# endpoints the script calls and applies the requested jq expression.
FAKE_GH = """#!/usr/bin/env bash
set -euo pipefail
ENDPOINT=""
EXPR="."
while [ $# -gt 0 ]; do
    case "$1" in
        api | --paginate) shift ;;
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
    */pulls/*/files*) FILE="$FAKE_GH_DIR/files.json" ;;
    */pulls/*/reviews*) FILE="$FAKE_GH_DIR/reviews.json" ;;
    */pulls/*) FILE="$FAKE_GH_DIR/pull.json" ;;
    *)
        echo "unexpected endpoint: $ENDPOINT" >&2
        exit 1
        ;;
esac
jq -r "$EXPR" "$FILE"
"""


class Harness:
    """A fake gh serving one pull request's labels, files, and reviews."""

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
        self.pull()
        self.files("console/src/app/App.tsx")
        self.reviews()

    def pull(self, *labels: str, author: str = "author") -> None:
        payload = {"user": {"login": author}, "labels": [{"name": l} for l in labels]}
        (self.data / "pull.json").write_text(json.dumps(payload))

    def files(self, *names: str) -> None:
        payload = [{"filename": name} for name in names]
        (self.data / "files.json").write_text(json.dumps(payload))

    def reviews(self, *reviews: tuple[str, str] | tuple[str, str, str]) -> None:
        """Sets the reviews in order as (login, state) or (login, state, user type)."""
        payload = [
            {
                "user": {"login": r[0], "type": r[2] if len(r) > 2 else "User"},
                "state": r[1],
            }
            for r in reviews
        ]
        (self.data / "reviews.json").write_text(json.dumps(payload))

    def run(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT), "42"], env=self.env, capture_output=True, text=True
        )


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    return Harness(tmp_path)


class TestLabels:
    """Tests for the tier label requirement."""

    def test_fails_without_a_tier_label(self, harness: Harness) -> None:
        harness.pull("console")
        result = harness.run()
        assert result.returncode != 0
        assert "exactly one review tier label" in result.stderr
        assert "found: none" in result.stderr

    def test_fails_with_two_tier_labels(self, harness: Harness) -> None:
        harness.pull("review/light", "review/bot")
        result = harness.run()
        assert result.returncode != 0
        assert "found: review/light review/bot" in result.stderr

    def test_ignores_other_labels(self, harness: Harness) -> None:
        harness.pull("console", "review/bot", "do not merge")
        assert harness.run().returncode == 0


class TestFloor:
    """Tests for the floor that changed paths set."""

    def test_bot_passes_off_the_floor_paths(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.files("pluto/src/button/Button.tsx", "core/pkg/api/user.go")
        result = harness.run()
        assert result.returncode == 0, result.stderr
        assert "no human approval required" in result.stdout

    def test_bot_fails_under_schemas(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.files("docs/site/src/x.mdx", "schemas/synnax/channel.oracle")
        result = harness.run()
        assert result.returncode != 0
        assert "review/bot is below the floor review/light" in result.stderr
        assert "schemas/synnax/channel.oracle" in result.stderr

    def test_bot_passes_for_tests_under_a_floor_path(self, harness: Harness) -> None:
        harness.pull("review/bot")
        harness.files("cesium/db_test.go", "console/src/a.spec.tsx")
        assert harness.run().returncode == 0

    def test_light_satisfies_the_floor(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.files("aspen/internal/kv/gossip.go")
        harness.reviews(("reviewer", "APPROVED"))
        assert harness.run().returncode == 0


class TestApproval:
    """Tests for the human approval a tier needs."""

    def test_light_fails_without_an_approval(self, harness: Harness) -> None:
        harness.pull("review/light")
        result = harness.run()
        assert result.returncode != 0
        assert "review/light needs an approval from someone other than author" in (
            result.stderr
        )

    def test_thorough_passes_with_one_approval(self, harness: Harness) -> None:
        harness.pull("review/thorough")
        harness.reviews(("reviewer", "APPROVED"))
        result = harness.run()
        assert result.returncode == 0, result.stderr
        assert "review/thorough: approved by reviewer" in result.stdout

    def test_ignores_the_author(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("author", "APPROVED"))
        assert harness.run().returncode != 0

    def test_ignores_bots(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("greptile-apps[bot]", "APPROVED", "Bot"))
        assert harness.run().returncode != 0

    def test_a_later_request_for_changes_retires_an_approval(
        self, harness: Harness
    ) -> None:
        harness.pull("review/light")
        harness.reviews(("reviewer", "APPROVED"), ("reviewer", "CHANGES_REQUESTED"))
        assert harness.run().returncode != 0

    def test_a_comment_does_not_retire_an_approval(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("reviewer", "APPROVED"), ("reviewer", "COMMENTED"))
        assert harness.run().returncode == 0

    def test_a_dismissed_approval_does_not_count(self, harness: Harness) -> None:
        harness.pull("review/light")
        harness.reviews(("reviewer", "DISMISSED"))
        assert harness.run().returncode != 0
