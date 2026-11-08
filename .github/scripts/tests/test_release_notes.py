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

import pytest
from scripts_common import FakeGH

SCRIPT = Path(__file__).resolve().parents[1] / "release_notes.sh"

GENERATED = """## What's Changed
### Driver
* Fix a read by @someone in https://github.com/synnaxlabs/synnax/pull/1

## New Contributors
* @newcomer made their first contribution in https://github.com/synnaxlabs/synnax/pull/2

**Full Changelog**: https://github.com/synnaxlabs/synnax/compare/v0.1.0...v0.1.1
"""


class Harness:
    """A fake gh serving one generate-notes response."""

    def __init__(self, *, path: Path, body: str = GENERATED) -> None:
        self.gh = FakeGH(path)
        self.gh.serve("api */releases/generate-notes", {"body": body})
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def run(self, *args: str) -> str:
        """Runs the script and returns the notes it prints."""
        result = subprocess.run(
            [str(SCRIPT), *args], env=self.env, capture_output=True, text=True
        )
        assert result.returncode == 0, result.stderr
        return result.stdout

    def gh_args(self) -> list[str]:
        """Returns the arguments the script passed to gh."""
        return self.gh.calls()[-1]


@pytest.fixture
def harness(tmp_path: Path) -> Harness:
    return Harness(path=tmp_path)


class TestReleaseNotes:
    def test_generates_from_the_config(self, harness: Harness) -> None:
        harness.run("driver/v0.1.1", "abc", ".github/release/driver.yml")
        args = harness.gh_args()
        assert args[:2] == ["api", "repos/synnaxlabs/synnax/releases/generate-notes"]
        assert "tag_name=driver/v0.1.1" in args
        assert "target_commitish=abc" in args
        assert "configuration_file_path=.github/release/driver.yml" in args

    def test_starts_after_the_previous_tag(self, harness: Harness) -> None:
        harness.run("driver/v0.1.1", "abc", "c.yml", "driver/v0.1.0")
        assert "previous_tag_name=driver/v0.1.0" in harness.gh_args()

    def test_omits_the_previous_tag_for_a_first_release(self, harness: Harness) -> None:
        harness.run("driver/v0.1.0", "abc", "c.yml")
        assert not any(a.startswith("previous_tag_name") for a in harness.gh_args())

    def test_drops_new_contributors_and_keeps_the_changelog(
        self, harness: Harness
    ) -> None:
        notes = harness.run("driver/v0.1.1", "abc", "c.yml")
        assert "New Contributors" not in notes
        assert "@newcomer" not in notes
        assert "* Fix a read by @someone" in notes
        assert "**Full Changelog**" in notes

    def test_puts_the_given_notes_first(self, harness: Harness) -> None:
        notes = harness.run("core/v0.1.1", "abc", "c.yml", "", "Embeds Console v0.1.1.")
        assert notes.startswith("Embeds Console v0.1.1.\n\n## What's Changed")

    def test_prints_only_the_generated_notes_without_given_notes(
        self, harness: Harness
    ) -> None:
        notes = harness.run("driver/v0.1.1", "abc", "c.yml")
        assert notes.startswith("## What's Changed")
