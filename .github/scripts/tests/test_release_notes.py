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
PULL = "https://github.com/synnaxlabs/synnax/pull"

GENERATED = f"""## What's Changed
### Driver
* Fix a read by @someone in {PULL}/1

## New Contributors
* @newcomer made their first contribution in {PULL}/2

**Full Changelog**: https://github.com/synnaxlabs/synnax/compare/v0.1.0...v0.1.1
"""


class Harness:
    """A scratch Git repository and a fake gh serving one generate-notes response."""

    def __init__(self, *, path: Path, body: str = GENERATED) -> None:
        self.path = path / "repo"
        self.path.mkdir()
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.email", "ci@synnaxlabs.com")
        self.git("config", "user.name", "ci")
        self.files = 0
        self.commit("root")
        self.gh = FakeGH(path)
        self.gh.serve("api */releases/generate-notes", {"body": body})
        self.env = {
            **os.environ,
            "PATH": f"{self.gh.bin_dir}:{os.environ['PATH']}",
            "GITHUB_REPOSITORY": "synnaxlabs/synnax",
        }

    def git(self, *args: str) -> str:
        return subprocess.run(
            ["git", *args], cwd=self.path, check=True, capture_output=True, text=True
        ).stdout.strip()

    def commit(self, message: str) -> str:
        """Commits a new file and returns the commit's SHA."""
        self.files += 1
        (self.path / f"file{self.files}").write_text(message)
        self.git("add", "-A")
        self.git("commit", "-q", "-m", message)
        return self.git("rev-parse", "HEAD")

    def run(self, *args: str) -> str:
        """Runs the script and returns the notes it prints."""
        result = subprocess.run(
            [str(SCRIPT), *args],
            cwd=self.path,
            env=self.env,
            capture_output=True,
            text=True,
        )
        assert result.returncode == 0, result.stderr
        return result.stdout

    def gh_args(self) -> list[str]:
        """Returns the arguments the script passed to generate the notes."""
        return next(c for c in self.gh.calls() if c[1].endswith("/generate-notes"))


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

    def test_fails_in_a_shallow_clone(self, harness: Harness, tmp_path: Path) -> None:
        harness.commit("more")
        shallow = tmp_path / "shallow"
        subprocess.run(
            ["git", "clone", "-q", "--depth", "1", f"file://{harness.path}", shallow],
            check=True,
        )
        result = subprocess.run(
            [str(SCRIPT), "core/v0.1.1", "HEAD", "c.yml"],
            cwd=shallow,
            env=harness.env,
            capture_output=True,
            text=True,
        )
        assert result.returncode != 0
        assert "full clone" in result.stderr


MINOR = f"""## What's Changed
### Core
* Fix a stall by @someone in {PULL}/11
* Fix a leak by @someone in {PULL}/12
### Arc
* Fix a stage by @someone in {PULL}/14


**Full Changelog**: https://github.com/synnaxlabs/synnax/compare/v0.60.1...v0.61.0
"""


class Train(Harness):
    """Main holds a stall fix (#11), a leak fix (#12), and a stage fix (#14) after
    core/v0.60.0. The checkout is release/core-0.60 at core/v0.60.0."""

    def __init__(self, *, path: Path) -> None:
        super().__init__(path=path, body=MINOR)
        self.git("tag", "core/v0.60.0")
        self.stall = self.commit("Fix a stall (#11)")
        self.commit("Fix a leak (#12)")
        self.stage = self.commit("Fix a stage (#14)")
        self.git("checkout", "-q", "-b", "release/core-0.60", "core/v0.60.0")
        # A pick that keeps its subject names the source PR, whose commits hold no
        # trailers.
        self.gh.serve("api */pulls/1[124]/commits")


@pytest.fixture
def train(tmp_path: Path) -> Train:
    return Train(path=tmp_path)


def picked(subject: str, sha: str) -> dict[str, dict[str, str]]:
    """Returns a PR commit that `cherry-pick -x` made from sha."""
    return {"commit": {"message": f"{subject}\n\n(cherry picked from commit {sha})"}}


class TestCherryPicks:
    """Tests for leaving out PRs that an earlier patch release shipped."""

    def test_leaves_out_fixes_picked_in_a_squash_merged_pr(self, train: Train) -> None:
        train.commit("Pick the stall and stage fixes (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/13/commits",
            [
                picked("Fix a stall (#11)", train.stall),
                picked("Fix a stage (#14)", train.stage),
            ],
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"{PULL}/14" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_leaves_out_a_fix_picked_with_its_trailer(self, train: Train) -> None:
        train.git("cherry-pick", "-x", train.stall)
        train.git("tag", "core/v0.60.1")
        notes = train.run("core/v0.61.0-rc.1", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes
        assert f"* Fix a stage by @someone in {PULL}/14" in notes

    def test_keeps_every_fix_without_a_patch_release(self, train: Train) -> None:
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.0")
        assert notes.strip() == MINOR.strip()

    def test_leaves_out_a_category_whose_fixes_all_shipped(self, train: Train) -> None:
        train.git("cherry-pick", "-x", train.stage)
        train.git("tag", "core/v0.60.1")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert "### Arc" not in notes
        assert f"### Core\n* Fix a stall by @someone in {PULL}/11" in notes

    def test_keeps_a_fix_the_previous_tag_does_not_hold(self, train: Train) -> None:
        train.git("checkout", "-q", "-b", "release/core-0.59", "main~3")
        train.git("cherry-pick", "-x", train.stall)
        train.git("tag", "core/v0.59.2")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.0")
        assert f"* Fix a stall by @someone in {PULL}/11" in notes

    def test_leaves_out_a_fix_picked_from_its_pr_branch(self, train: Train) -> None:
        branch = "0123456789abcdef0123456789abcdef01234567"
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */pulls/13/commits", [picked("Fix a stall", branch)])
        train.gh.serve(
            f"api */commits/{branch}/pulls",
            [{"number": 11, "merged_at": "2026-11-01T00:00:00Z"}],
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a stage by @someone in {PULL}/14" in notes
