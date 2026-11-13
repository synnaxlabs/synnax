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
from typing import Any

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "clear_release_drafts.sh"
TAG = "core/v0.59.3"
RELEASES = "repos/synnaxlabs/synnax/releases"

# Stands in for gh: records each call to FAKE_GH_CALLS and serves the pages in
# FAKE_GH_PAGES through the requested jq expression. A delete fails when
# FAKE_GH_DELETES is empty, and a listing fails when no pages exist.
FAKE_GH = """#!/usr/bin/env bash
set -euo pipefail
printf '%s\\n' "$*" >> "$FAKE_GH_CALLS"
if [ "${3:-}" = --method ]; then
    [ -n "$FAKE_GH_DELETES" ]
    exit
fi
jq -c ".[] | $5" "$FAKE_GH_PAGES"
"""


def release(id: int, tag: str = TAG, draft: bool = True) -> dict[str, Any]:
    return {"id": id, "tag_name": tag, "draft": draft, "name": "ignored"}


class Harness:
    """A fake gh serving pages of releases and accepting deletes by ID."""

    def __init__(self, *, path: Path, deletes_succeed: bool = True) -> None:
        bin_dir = path / "bin"
        bin_dir.mkdir()
        gh = bin_dir / "gh"
        gh.write_text(FAKE_GH)
        gh.chmod(0o755)
        self.calls_file = path / "calls"
        self.calls_file.touch()
        self.pages_file = path / "pages.json"
        self.env = {
            **os.environ,
            "PATH": f"{bin_dir}:{os.environ['PATH']}",
            "FAKE_GH_CALLS": str(self.calls_file),
            "FAKE_GH_PAGES": str(self.pages_file),
            "FAKE_GH_DELETES": "true" if deletes_succeed else "",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def serve(self, *pages: list[dict[str, Any]]) -> None:
        self.pages_file.write_text(json.dumps(pages))

    def run(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT), TAG], env=self.env, capture_output=True, text=True
        )

    def calls(self) -> list[list[str]]:
        """Returns the arguments of each call to gh, oldest first."""
        return [line.split() for line in self.calls_file.read_text().splitlines()]

    def deleted(self) -> list[str]:
        """Returns the path of each delete the script made."""
        return [c[1] for c in self.calls() if "DELETE" in c]


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    return Harness(path=tmp_path)


class TestClearReleaseDrafts:
    def test_deletes_a_draft_by_id(self, harness: Harness) -> None:
        harness.serve([release(1, "core/v0.60.1", draft=False), release(7)])
        assert harness.run().returncode == 0
        assert harness.deleted() == [f"{RELEASES}/7"]

    def test_deletes_every_draft_under_the_tag(self, harness: Harness) -> None:
        harness.serve([release(7)], [release(8)])
        assert harness.run().returncode == 0
        assert harness.deleted() == [f"{RELEASES}/7", f"{RELEASES}/8"]

    def test_finds_a_draft_past_the_first_page(self, harness: Harness) -> None:
        harness.serve([release(1, "core/v0.60.1", draft=False)], [release(7)])
        assert harness.run().returncode == 0
        assert harness.deleted() == [f"{RELEASES}/7"]
        assert "--paginate" in harness.calls()[0]

    def test_leaves_drafts_of_other_tags(self, harness: Harness) -> None:
        harness.serve([release(7, "core/v0.60.2")])
        assert harness.run().returncode == 0
        assert harness.deleted() == []

    def test_does_nothing_when_no_release_holds_the_tag(self, harness: Harness) -> None:
        harness.serve([])
        assert harness.run().returncode == 0
        assert harness.deleted() == []

    def test_fails_when_a_published_release_holds_the_tag(
        self, harness: Harness
    ) -> None:
        harness.serve([release(5, draft=False)])
        result = harness.run()
        assert result.returncode == 1
        assert f"published release 5 already holds {TAG}" in result.stderr
        assert harness.deleted() == []

    def test_keeps_a_draft_that_shares_the_tag_with_a_published_release(
        self, harness: Harness
    ) -> None:
        harness.serve([release(7), release(5, draft=False)])
        assert harness.run().returncode == 1
        assert harness.deleted() == []

    def test_fails_without_deleting_when_the_listing_fails(
        self, harness: Harness
    ) -> None:
        assert harness.run().returncode != 0
        assert harness.deleted() == []

    def test_fails_when_a_delete_fails(self, tmp_path: Path) -> None:
        harness = Harness(path=tmp_path, deletes_succeed=False)
        harness.serve([release(7)])
        assert harness.run().returncode != 0
