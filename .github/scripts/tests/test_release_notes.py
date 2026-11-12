#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import re
import subprocess
from pathlib import Path
from typing import Any

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
        harness.git("tag", "driver/v0.1.0")
        harness.run("driver/v0.1.1", "HEAD", "c.yml", "driver/v0.1.0")
        assert "previous_tag_name=driver/v0.1.0" in harness.gh_args()

    def test_fails_when_the_previous_tag_does_not_exist(self, harness: Harness) -> None:
        result = subprocess.run(
            [str(SCRIPT), "driver/v0.1.1", "HEAD", "c.yml", "driver/v0.1.0"],
            cwd=harness.path,
            env=harness.env,
            capture_output=True,
            text=True,
        )
        assert result.returncode != 0
        assert "driver/v0.1.0" in result.stderr
        assert not harness.gh.calls()

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

    def test_cuts_a_body_over_the_github_limit(self, tmp_path: Path) -> None:
        entries = "".join(
            f"* Fix read number {i} by @someone in {PULL}/{i}\n" for i in range(3000)
        )
        changelog = (
            "**Full Changelog**: https://github.com/synnaxlabs/synnax/compare/a...b"
        )
        body = f"## What's Changed\n### Driver\n{entries}\n{changelog}\n"
        assert len(body) > 125_000
        notes = Harness(path=tmp_path, body=body).run("driver/v0.1.1", "abc", "c.yml")
        assert len(notes) <= 125_000
        assert notes.startswith("## What's Changed\n### Driver\n* Fix read number 0 ")
        assert "\n* " in notes.split("_The list is cut short")[0][-200:]
        assert "\n\n\n" not in notes
        assert notes.endswith(f"{changelog}\n")
        assert all(
            re.fullmatch(rf"\* Fix read number (\d+) by @someone in {PULL}/\1", line)
            for line in notes.splitlines()
            if line.startswith("* ")
        )

    def test_ends_a_cut_list_with_an_entry(self, tmp_path: Path) -> None:
        changelog = (
            "**Full Changelog**: https://github.com/synnaxlabs/synnax/compare/a...b"
        )
        endings = set()
        for width in range(10, 60):
            category = f"### {'C' * width}\n* Fix a read by @someone in {PULL}/1\n"
            body = f"## What's Changed\n{category * 3000}\n{changelog}\n"
            path = tmp_path / str(width)
            path.mkdir()
            notes = Harness(path=path, body=body).run("driver/v0.1.1", "abc", "c.yml")
            kept = notes.split("\n\n_The list is cut short")[0].splitlines()
            assert kept[-1].startswith("* ")
            endings.add(kept[-2][:3])
        assert endings == {"###"}

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


WITH_REVERTS = MINOR.replace(
    f"{PULL}/12\n",
    f"{PULL}/12\n* Revert a stall fix by @someone in {PULL}/16\n"
    f"* Re-land the stall fix by @someone in {PULL}/17\n",
)

ONLY_STAGE = f"""## What's Changed
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
        self.leak = self.commit("Fix a leak (#12)")
        self.stage = self.commit("Fix a stage (#14)")
        self.git("checkout", "-q", "-b", "release/core-0.60", "core/v0.60.0")

    def serve_message(self, sha: str, message: str = "") -> None:
        """Serves the message of a commit that is not in the clone."""
        self.gh.serve(f"api */commits/{sha}", {"commit": {"message": message}})


@pytest.fixture
def train(tmp_path: Path) -> Train:
    return Train(path=tmp_path)


def pr_commit(sha: str, message: str = "", parents: int = 1) -> dict[str, Any]:
    """Returns a commit as the PR commits endpoint lists it."""
    return {"sha": sha, "parents": [{}] * parents, "commit": {"message": message}}


def picked(subject: str, source: str, sha: str = "f" * 40) -> dict[str, Any]:
    """Returns a PR commit that `cherry-pick -x` made from source."""
    return pr_commit(sha, f"{subject}\n\n(cherry picked from commit {source})")


def picked_revert(source: str, target: str, sha: str) -> dict[str, Any]:
    """Returns a PR commit that `cherry-pick -x` made from source, which reverts
    target."""
    return pr_commit(
        sha,
        f"Revert\n\nThis reverts commit {target}.\n\n"
        f"(cherry picked from commit {source})",
    )


MERGED_11 = [{"number": 11, "merged_at": "2026-11-01T00:00:00Z"}]


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
        train.serve_message(branch, "Fix a stall")
        train.gh.serve(f"api */commits/{branch}/pulls", MERGED_11)
        train.gh.serve("api */pulls/11/commits", [pr_commit(branch)])
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a stage by @someone in {PULL}/14" in notes

    def test_leaves_out_a_fix_picked_from_a_kept_pr_branch(self, train: Train) -> None:
        train.git("checkout", "-q", "-b", "fix-stall", "main~3")
        branch = train.commit("Fix a stall")
        train.git("checkout", "-q", "release/core-0.60")
        train.git("cherry-pick", "-x", branch)
        train.git("tag", "core/v0.60.1")
        train.gh.serve(f"api */commits/{branch}/pulls", MERGED_11)
        train.gh.serve("api */pulls/11/commits", [pr_commit(branch)])
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a stage by @someone in {PULL}/14" in notes

    def test_keeps_a_pr_whose_branch_was_picked_in_part(self, train: Train) -> None:
        branch = "0123456789abcdef0123456789abcdef01234567"
        later = "89abcdef0123456789abcdef0123456789abcdef"
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */pulls/13/commits", [picked("Fix a stall", branch)])
        train.serve_message(branch, "Fix a stall")
        train.gh.serve(f"api */commits/{branch}/pulls", MERGED_11)
        train.gh.serve("api */pulls/11/commits", [pr_commit(branch), pr_commit(later)])
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert notes.strip() == MINOR.strip()

    def test_leaves_out_a_fix_picked_from_an_older_train(self, train: Train) -> None:
        train.git("checkout", "-q", "-b", "release/core-0.59", "core/v0.60.0")
        older = train.commit("Pick the stall fix (0.59.2) (#20)")
        train.gh.serve(
            "api */pulls/20/commits",
            [picked("Fix a stall (#11)", train.stall, "a" * 40)],
        )
        train.git("checkout", "-q", "release/core-0.60")
        train.commit("Pick the stall fix (0.60.1) (#21)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/21/commits",
            [picked("Pick the stall fix (0.59.2) (#20)", older)],
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_reads_no_pr_commits_for_a_pick_with_a_trailer(self, train: Train) -> None:
        train.git("cherry-pick", "-x", train.stall)
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/11/commits", [picked("Fix a leak (#12)", train.leak)]
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_keeps_every_fix_when_github_lacks_the_source(self, train: Train) -> None:
        local = "0123456789abcdef0123456789abcdef01234567"
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */pulls/13/commits", [picked("Fix a stall", local)])
        for route in (f"api */commits/{local}", f"api */commits/{local}/pulls"):
            train.gh.fail(route, f"gh: No commit found for SHA: {local} (HTTP 422)")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert notes.strip() == MINOR.strip()

    def test_fails_when_github_rejects_the_source_lookup(self, train: Train) -> None:
        branch = "0123456789abcdef0123456789abcdef01234567"
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */pulls/13/commits", [picked("Fix a stall", branch)])
        train.serve_message(branch, "Fix a stall")
        train.gh.fail(
            f"api */commits/{branch}/pulls", "gh: Resource not accessible (HTTP 403)"
        )
        result = subprocess.run(
            [str(SCRIPT), "core/v0.61.0", "main", "c.yml", "core/v0.60.1"],
            cwd=train.path,
            env=train.env,
            capture_output=True,
            text=True,
        )
        assert result.returncode != 0
        assert "HTTP 403" in result.stderr

    def test_ignores_merge_commits_of_a_picked_pr_branch(self, train: Train) -> None:
        branch = "0123456789abcdef0123456789abcdef01234567"
        merge = "89abcdef0123456789abcdef0123456789abcdef"
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */pulls/13/commits", [picked("Fix a stall", branch)])
        train.serve_message(branch, "Fix a stall")
        train.gh.serve(f"api */commits/{branch}/pulls", MERGED_11)
        train.gh.serve(
            "api */pulls/11/commits", [pr_commit(branch), pr_commit(merge, parents=2)]
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes

    def test_keeps_a_fix_whose_pick_was_reverted(self, train: Train) -> None:
        train.git("cherry-pick", "-x", train.stall)
        train.git("revert", "--no-edit", "HEAD")
        train.git("tag", "core/v0.60.1")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert notes.strip() == MINOR.strip()

    def test_keeps_a_fix_reverted_inside_its_pick_pr(self, train: Train) -> None:
        stage_pick = "0123456789abcdef0123456789abcdef01234567"
        train.commit("Pick the stall and stage fixes (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/13/commits",
            [
                picked("Fix a stall (#11)", train.stall, "a" * 40),
                picked("Fix a stage (#14)", train.stage, stage_pick),
                pr_commit("b" * 40, f"Revert\n\nThis reverts commit {stage_pick}."),
            ],
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a stage by @someone in {PULL}/14" in notes

    def test_keeps_the_fixes_of_a_reverted_pick_pr(self, train: Train) -> None:
        pick = train.commit("Pick the stall and stage fixes (0.60.1) (#13)")
        train.commit('Revert "Pick the stall and stage fixes (0.60.1)" (#15)')
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/13/commits",
            [
                picked("Fix a stall (#11)", train.stall),
                picked("Fix a stage (#14)", train.stage),
            ],
        )
        train.gh.serve(
            "api */pulls/15/commits",
            [pr_commit("c" * 40, f"Revert\n\nThis reverts commit {pick}.")],
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert notes.strip() == MINOR.strip()

    def test_leaves_out_a_fix_whose_revert_was_reverted(self, train: Train) -> None:
        train.git("cherry-pick", "-x", train.stall)
        train.git("revert", "--no-edit", "HEAD")
        train.git("revert", "--no-edit", "HEAD")
        train.git("tag", "core/v0.60.1")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_keeps_a_fix_whose_revert_from_main_was_picked(self, train: Train) -> None:
        train.git("checkout", "-q", "-b", "later", "main")
        train.git("revert", "--no-edit", train.stall)
        revert = train.git("rev-parse", "HEAD")
        train.git("checkout", "-q", "release/core-0.60")
        train.git("cherry-pick", "-x", train.stall)
        train.git("cherry-pick", "-x", revert)
        train.git("tag", "core/v0.60.1")
        train.gh.serve(f"api */commits/{revert}/pulls")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"* Fix a stall by @someone in {PULL}/11" in notes

    def test_leaves_out_a_fix_whose_picked_revert_was_relanded(
        self, train: Train
    ) -> None:
        train.git("checkout", "-q", "main")
        train.git("revert", "--no-edit", train.stall)
        revert = train.git("rev-parse", "HEAD")
        train.git("revert", "--no-edit", revert)
        reland = train.git("rev-parse", "HEAD")
        train.git("checkout", "-q", "release/core-0.60")
        train.git("cherry-pick", "-x", train.stall)
        train.git("cherry-pick", "-x", revert)
        train.git("cherry-pick", "-x", reland)
        train.git("tag", "core/v0.60.1")
        train.gh.serve(f"api */commits/{revert}/pulls")
        train.gh.serve(f"api */commits/{reland}/pulls")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_leaves_out_a_fix_picked_again_after_its_revert(self, train: Train) -> None:
        train.git("checkout", "-q", "main")
        train.git("revert", "--no-edit", train.stall)
        revert = train.git("rev-parse", "HEAD")
        train.git("checkout", "-q", "release/core-0.60")
        train.git("cherry-pick", "-x", train.stall)
        train.git("cherry-pick", "-x", revert)
        train.git("cherry-pick", "-x", train.stall)
        train.git("tag", "core/v0.60.1")
        train.gh.serve(f"api */commits/{revert}/pulls")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes

    def test_keeps_a_fix_and_its_revert_picked_in_one_pr(self, train: Train) -> None:
        revert = "0123456789abcdef0123456789abcdef01234567"
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/13/commits",
            [
                picked("Fix a stall (#11)", train.stall, "a" * 40),
                picked_revert(revert, train.stall, "b" * 40),
            ],
        )
        train.serve_message(revert, f"Revert\n\nThis reverts commit {train.stall}.")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"* Fix a stall by @someone in {PULL}/11" in notes

    def test_leaves_out_a_fix_relanded_in_one_pick_pr(self, train: Train) -> None:
        revert = "0123456789abcdef0123456789abcdef01234567"
        reland = "89abcdef0123456789abcdef0123456789abcdef"
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/13/commits",
            [
                picked("Fix a stall (#11)", train.stall, "a" * 40),
                picked_revert(revert, train.stall, "b" * 40),
                picked_revert(reland, revert, "c" * 40),
            ],
        )
        train.serve_message(revert, f"Revert\n\nThis reverts commit {train.stall}.")
        train.serve_message(reland, f"Revert\n\nThis reverts commit {revert}.")
        train.gh.serve(f"api */commits/{revert}/pulls")
        train.gh.serve(f"api */commits/{reland}/pulls")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_fails_when_a_pick_pr_reaches_the_commit_limit(self, train: Train) -> None:
        train.commit("Pick the stall fix (0.60.1) (#13)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/13/commits",
            [picked("Fix a stall (#11)", train.stall, f"{i:040x}") for i in range(250)],
        )
        result = subprocess.run(
            [str(SCRIPT), "core/v0.61.0", "main", "c.yml", "core/v0.60.1"],
            cwd=train.path,
            env=train.env,
            capture_output=True,
            text=True,
        )
        assert result.returncode != 0
        assert "PR #13 has 250 or more commits" in result.stderr

    def test_keeps_a_fix_and_its_picked_revert_pr(self, train: Train) -> None:
        train.git("checkout", "-q", "main")
        revert = train.commit(
            f'Revert "Fix a stall (#11)" (#16)\n\nThis reverts commit {train.stall}.'
        )
        train.git("checkout", "-q", "release/core-0.60")
        train.git("cherry-pick", "-x", train.stall)
        train.git("cherry-pick", "-x", revert)
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */releases/generate-notes", {"body": WITH_REVERTS})
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11\n" in notes
        assert f"{PULL}/16\n" in notes

    def test_leaves_out_a_fix_and_its_relanded_revert_prs(self, train: Train) -> None:
        train.git("checkout", "-q", "main")
        revert = train.commit(
            f'Revert "Fix a stall (#11)" (#16)\n\nThis reverts commit {train.stall}.'
        )
        reland = train.commit(
            f"Re-land the stall fix (#17)\n\nThis reverts commit {revert}."
        )
        train.git("checkout", "-q", "release/core-0.60")
        for commit in (train.stall, revert, reland):
            train.git("cherry-pick", "-x", commit)
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */releases/generate-notes", {"body": WITH_REVERTS})
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        for pr in (11, 16, 17):
            assert f"{PULL}/{pr}\n" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_reads_the_commits_of_a_pick_pr_whose_message_has_trailers(
        self, train: Train
    ) -> None:
        stage_pick = "0123456789abcdef0123456789abcdef01234567"
        train.commit(
            "Pick the stall and stage fixes (0.60.1) (#13)\n\n"
            f"(cherry picked from commit {train.stall})\n"
            f"(cherry picked from commit {train.stage})\n"
            f"This reverts commit {stage_pick}."
        )
        train.git("tag", "core/v0.60.1")
        train.gh.serve(
            "api */pulls/13/commits",
            [
                picked("Fix a stall (#11)", train.stall, "a" * 40),
                picked("Fix a stage (#14)", train.stage, stage_pick),
                pr_commit("b" * 40, f"Revert\n\nThis reverts commit {stage_pick}."),
            ],
        )
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a stage by @someone in {PULL}/14" in notes

    def test_leaves_out_the_section_when_every_fix_shipped(self, train: Train) -> None:
        train.git("cherry-pick", "-x", train.stage)
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */releases/generate-notes", {"body": ONLY_STAGE})
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert "What's Changed" not in notes
        assert "**Full Changelog**" in notes

    def test_follows_a_pick_from_a_commit_outside_the_clone(self, train: Train) -> None:
        branch = "0123456789abcdef0123456789abcdef01234567"
        train.commit("Pick the stall fix (0.60.1) (#21)")
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */pulls/21/commits", [picked("Fix a stall", branch)])
        train.serve_message(
            branch, f"Fix a stall\n\n(cherry picked from commit {train.stall})"
        )
        train.gh.serve(f"api */commits/{branch}/pulls")
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/11" not in notes
        assert f"* Fix a leak by @someone in {PULL}/12" in notes

    def test_leaves_out_a_picked_revert_of_a_fix_from_the_base(
        self, train: Train
    ) -> None:
        base = train.git("rev-parse", "core/v0.60.0")
        train.git("checkout", "-q", "main")
        revert = train.commit(
            f"Revert a stall fix (#16)\n\nThis reverts commit {base}."
        )
        train.git("checkout", "-q", "release/core-0.60")
        train.git("cherry-pick", "-x", revert)
        train.git("tag", "core/v0.60.1")
        train.gh.serve("api */releases/generate-notes", {"body": WITH_REVERTS})
        notes = train.run("core/v0.61.0", "main", "c.yml", "core/v0.60.1")
        assert f"{PULL}/16\n" not in notes
        assert f"* Fix a stall by @someone in {PULL}/11" in notes
