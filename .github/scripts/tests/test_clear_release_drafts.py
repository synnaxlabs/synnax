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
from typing import Any

import pytest
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "clear_release_drafts.sh"
TAG = "core/v0.59.3"
RELEASES = "repos/synnaxlabs/synnax/releases"


def release(id: int, tag: str = TAG, draft: bool = True) -> dict[str, Any]:
    return {"id": id, "tag_name": tag, "draft": draft, "name": "ignored"}


class Harness:
    """A fake gh serving pages of releases and accepting deletes by ID."""

    def __init__(self, *, path: Path, deletes_succeed: bool = True) -> None:
        self.gh = FakeGH(path)
        if deletes_succeed:
            self.gh.serve(f"api {RELEASES}/*")
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def serve(self, *pages: list[dict[str, Any]]) -> None:
        self.gh.serve(f"api {RELEASES}", *pages)

    def run(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [str(SCRIPT), TAG], env=self.env, capture_output=True, text=True
        )

    def deleted(self) -> list[str]:
        """Returns the path of each delete the script made."""
        return [c[1] for c in self.gh.calls() if "DELETE" in c]


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
        assert "--paginate" in harness.gh.calls()[0]

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
