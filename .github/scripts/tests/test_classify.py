#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import sys
from pathlib import Path
from typing import Any

import pytest

import classify
from classify import (
    BOT,
    HUMAN,
    MARKER,
    Surface,
    fence,
    needs_human,
    plan_labels,
    read_surfaces,
    render,
    trust_reasons,
)


def same(name: str) -> Surface:
    return Surface(name=name, status="same", diff="")


ALL_SAME = [same(n) for n in ("oracle", "client-ts", "client-py", "core", "driver")]


def pull(head_repo: str = "synnaxlabs/synnax") -> dict[str, Any]:
    return {
        "head": {"repo": {"full_name": head_repo}},
        "base": {"repo": {"full_name": "synnaxlabs/synnax"}},
    }


def file(name: str, *, patch: str | None = "+x") -> dict[str, Any]:
    return {"filename": name, "status": "modified", "patch": patch}


class TestReadSurfaces:
    """Tests reading the uploaded surface results."""

    def test_reads_status_and_diff(self, tmp_path: Path) -> None:
        (tmp_path / "jobs").write_text("")
        d = tmp_path / "surface-oracle"
        d.mkdir()
        (d / "status").write_text("changed\n")
        (d / "diff").write_text("+a\n")
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["oracle"] == Surface(
            name="oracle", status="changed", diff="+a\n"
        )

    def test_treats_a_missing_report_as_an_error(self, tmp_path: Path) -> None:
        (tmp_path / "jobs").write_text("Compare core\n")
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["core"].status == "error"

    def test_skips_a_surface_the_run_has_no_job_for(self, tmp_path: Path) -> None:
        (tmp_path / "jobs").write_text("Compare core\n")
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["core-windows"].status == "skipped"

    def test_treats_a_run_with_no_surface_jobs_as_an_error(
        self, tmp_path: Path
    ) -> None:
        (tmp_path / "jobs").write_text("Compare ${{ matrix.surface }}\n")
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["core"].status == "error"

    def test_reads_a_skipped_report(self, tmp_path: Path) -> None:
        (tmp_path / "jobs").write_text("Compare core\n")
        d = tmp_path / "surface-core"
        d.mkdir()
        (d / "status").write_text("skipped\n")
        surfaces = {s.name: s for s in read_surfaces(tmp_path)}
        assert surfaces["core"].status == "skipped"


class TestNeedsHuman:
    """Tests the decision from reports and reasons."""

    def test_is_false_when_nothing_changed(self) -> None:
        assert not needs_human(ALL_SAME, reasons=[])

    def test_is_true_when_a_surface_changed_or_failed(self) -> None:
        assert needs_human(
            [*ALL_SAME, Surface(name="core", status="changed", diff="")],
            reasons=[],
        )
        assert needs_human(
            [*ALL_SAME, Surface(name="core", status="error", diff="")],
            reasons=[],
        )

    def test_is_false_when_a_surface_was_skipped(self) -> None:
        skipped = Surface(name="core", status="skipped", diff="")
        assert not needs_human([*ALL_SAME, skipped], reasons=[])

    def test_is_true_when_a_reason_applies(self) -> None:
        assert needs_human(ALL_SAME, reasons=["fork"])


class TestPlanLabels:
    """Tests that the classifier only raises a tier."""

    def test_sets_human_and_removes_bot(self) -> None:
        assert plan_labels({BOT}, human=True) == ([HUMAN], [BOT])

    def test_sets_no_tier_when_none_is_set(self) -> None:
        assert plan_labels(set(), human=False) == ([], [])

    def test_never_removes_human(self) -> None:
        assert plan_labels({HUMAN}, human=False) == ([], [])
        assert plan_labels({HUMAN, BOT}, human=False) == ([], [BOT])

    def test_treats_old_human_tiers_as_human(self) -> None:
        assert plan_labels({"review/light"}, human=False) == ([HUMAN], [])
        assert plan_labels({"review/thorough", BOT}, human=False) == ([HUMAN], [BOT])

    def test_changes_nothing_when_the_label_is_right(self) -> None:
        assert plan_labels({HUMAN}, human=True) == ([], [])
        assert plan_labels({BOT}, human=False) == ([], [])


class TestTrustReasons:
    """Tests the cases where the reports could be forged."""

    def test_is_empty_for_a_branch_that_leaves_the_classifier_alone(self) -> None:
        assert trust_reasons(pull(), files=[file("core/main.go")]) == []

    def test_flags_a_fork(self) -> None:
        assert trust_reasons(pull("someone/synnax"), files=[]) == [
            "The pull request comes from a fork."
        ]

    def test_flags_a_change_to_a_generator(self) -> None:
        reasons = trust_reasons(pull(), files=[file("scripts/surface_py.py")])
        assert reasons == [
            "The pull request changes the classifier: scripts/surface_py.py."
        ]


class TestRender:
    """Tests the sticky comment."""

    def test_shows_the_tier_and_the_changed_diff(self) -> None:
        surfaces = [
            *ALL_SAME[:4],
            Surface(name="driver", status="changed", diff="+flag --x\n"),
        ]
        body = render(surfaces, reasons=[], kept=False)
        assert body.startswith(MARKER)
        assert f"`{HUMAN}`" in body
        assert "| Driver commands and config keys | Changed |" in body
        assert "+flag --x" in body

    def test_explains_a_kept_human_label(self) -> None:
        body = render(ALL_SAME, reasons=[], kept=True)
        assert f"## Review tier: `{HUMAN}`" in body
        assert "already set" in body

    def test_lists_trust_reasons(self) -> None:
        body = render(
            ALL_SAME,
            reasons=["The pull request comes from a fork."],
            kept=False,
        )
        assert f"## Review tier: `{HUMAN}`" in body
        assert "- The pull request comes from a fork." in body

    def test_says_the_author_picks_when_nothing_changed(self) -> None:
        assert "## Review tier: the author picks" in render(
            ALL_SAME, reasons=[], kept=False
        )

    def test_shows_a_skipped_surface_without_raising_the_tier(self) -> None:
        skipped = Surface(name="core", status="skipped", diff="")
        body = render([*ALL_SAME, skipped], reasons=[], kept=False)
        assert "## Review tier: the author picks" in body
        assert "Not compared" in body
        assert "<details>" not in body

    def test_fences_a_diff_that_holds_backticks(self) -> None:
        assert fence("+```\n").startswith("````diff")


REPO = "synnaxlabs/synnax"
LABELS = f"repos/{REPO}/issues/5/labels"
COMMENTS = f"repos/{REPO}/issues/5/comments"


class FakeGH:
    """Serves pull request 5 through the gh API calls the classifier makes."""

    def __init__(self) -> None:
        self.heads = ["abc", "abc"]
        self.files = ["core/main.go"]
        self.labels: list[str] = []
        self.comments: list[dict[str, Any]] = []
        self.writes: list[tuple[tuple[str, ...], str | None]] = []

    def __call__(self, *args: str, stdin: str | None = None) -> str:
        if args == ("api", f"repos/{REPO}/pulls/5"):
            same = {"repo": {"full_name": REPO}}
            head = {**same, "sha": self.heads.pop(0)}
            labels = [{"name": n} for n in self.labels]
            return json.dumps({"head": head, "base": same, "labels": labels})
        if f"repos/{REPO}/pulls/5/files" in args:
            return "".join(json.dumps(file(name)) + "\n" for name in self.files)
        if args[:3] == ("api", "--paginate", COMMENTS):
            return "".join(json.dumps(c) + "\n" for c in self.comments)
        self.writes.append((args, stdin))
        return ""


class TestMain:
    """Tests labeling and commenting on a pull request end to end."""

    @pytest.fixture
    def gh(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> FakeGH:
        (tmp_path / "pr").mkdir()
        (tmp_path / "pr" / "number").write_text("5\n")
        jobs = "".join(f"Compare {name}\n" for name in classify.SURFACES)
        (tmp_path / "jobs").write_text(jobs)
        for name in classify.SURFACES:
            d = tmp_path / f"surface-{name}"
            d.mkdir()
            (d / "status").write_text("same\n")
        fake = FakeGH()
        monkeypatch.setattr(classify, "gh", fake)
        monkeypatch.setattr(sys, "argv", ["classify.py", str(tmp_path)])
        monkeypatch.setenv("GITHUB_REPOSITORY", REPO)
        monkeypatch.setenv("HEAD_SHA", "abc")
        return fake

    def test_keeps_the_tier_of_a_branch_older_than_a_report(
        self, gh: FakeGH, tmp_path: Path
    ) -> None:
        gh.labels = [BOT]
        (tmp_path / "surface-core" / "status").write_text("skipped\n")
        classify.main()
        assert not any(LABELS in args for args, _ in gh.writes)
        assert "Not compared" in str(gh.writes[-1][1])

    def test_raises_when_a_report_is_missing(self, gh: FakeGH, tmp_path: Path) -> None:
        (tmp_path / "surface-core" / "status").unlink()
        classify.main()
        assert any(LABELS in args for args, _ in gh.writes)

    def test_adds_no_label_and_posts_a_comment_when_nothing_changed(
        self, gh: FakeGH
    ) -> None:
        classify.main()
        ((post, body),) = gh.writes
        assert post == ("api", "-X", "POST", COMMENTS, "--input", "-")
        assert body is not None
        assert "the author picks" in json.loads(body)["body"]

    def test_swaps_bot_for_human_and_edits_the_comment_when_a_surface_changed(
        self, gh: FakeGH, tmp_path: Path
    ) -> None:
        (tmp_path / "surface-core" / "status").write_text("changed\n")
        gh.labels = [BOT]
        gh.comments = [{"id": 9, "body": f"{MARKER}\nold"}]
        classify.main()
        assert [args for args, _ in gh.writes] == [
            ("api", "-X", "POST", LABELS, "-f", f"labels[]={HUMAN}"),
            ("api", "-X", "DELETE", f"{LABELS}/review%2Fbot"),
            (
                "api",
                "-X",
                "PATCH",
                f"repos/{REPO}/issues/comments/9",
                "--input",
                "-",
            ),
        ]

    def test_adds_no_label_when_only_the_ui_changed(self, gh: FakeGH) -> None:
        gh.files = ["console/a.tsx"]
        classify.main()
        assert [args[3] for args, _ in gh.writes] == [COMMENTS]

    def test_keeps_the_tier_the_author_set_when_nothing_changed(
        self, gh: FakeGH
    ) -> None:
        gh.labels = [BOT]
        classify.main()
        assert [args[3] for args, _ in gh.writes] == [COMMENTS]

    def test_reads_the_number_from_the_environment_first(
        self, gh: FakeGH, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        (tmp_path / "pr" / "number").write_text("99\n")
        monkeypatch.setenv("PR", "5")
        classify.main()
        ((post, _),) = gh.writes
        assert post == ("api", "-X", "POST", COMMENTS, "--input", "-")

    def test_leaves_an_unchanged_comment_alone(self, gh: FakeGH) -> None:
        classify.main()
        ((_, body),) = gh.writes
        assert body is not None
        gh.comments = [{"id": 9, "body": json.loads(body)["body"]}]
        gh.writes = []
        gh.heads = ["abc", "abc"]
        classify.main()
        assert gh.writes == []

    def test_writes_nothing_when_the_head_moves_during_the_run(
        self, gh: FakeGH
    ) -> None:
        gh.heads = ["abc", "def"]
        classify.main()
        assert gh.writes == []
