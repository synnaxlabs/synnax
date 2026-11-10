#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import subprocess
from pathlib import Path
from typing import Any

import pytest

from changed_jobs import FILTERS, WORKFLOW, load
from job_key import Entry, glob_regex, keys, reusable, reused, tree

CI = """\
jobs:
  changes:
    runs-on: ubuntu-latest
  core-image:
    needs: changes
    if: needs.changes.outputs.core == 'true'
    uses: ./.github/workflows/build.yaml
  core:
    needs: changes
    if: needs.changes.outputs.core == 'true' ||
      contains(fromJSON(needs.changes.outputs.ci), 'core')
    uses: ./.github/workflows/test.yaml
  client:
    needs: [changes, core-image]
    if: needs.changes.outputs.client == 'true' || needs.changes.outputs.core == 'true'
    uses: ./.github/workflows/test.yaml
    with:
      tidy: ${{ needs.changes.outputs.client == 'true' }}
  copyright:
    runs-on: ubuntu-latest
  ok:
    needs: [changes, client, copyright, core, core-image]
    runs-on: ubuntu-latest
"""

FILTERS_TEXT = """\
go: &go
  - .golangci.yaml
core: &core
  - *go
  - core/**
client:
  - *core
  - client/**
  - "**/*.arc"
"""

WORKFLOWS = {
    ".github/workflows/build.yaml": "jobs: {}\n",
    ".github/workflows/test.yaml": (
        "jobs:\n  a:\n    uses: ./.github/workflows/inner.yaml\n"
    ),
    ".github/workflows/inner.yaml": "jobs: {}\n",
}

ENTRIES: list[Entry] = [
    (".github/scripts/a.py", "s1"),
    (".github/workflows/build.yaml", "w1"),
    (".github/workflows/inner.yaml", "w2"),
    (".github/workflows/test.yaml", "w3"),
    (".golangci.yaml", "g1"),
    ("client/a.ts", "c1"),
    ("core/a.go", "k1"),
    ("docs/a.md", "d1"),
]


def run(
    entries: list[Entry] = ENTRIES,
    *,
    ci: str = CI,
    outputs: dict[str, str] | None = None,
    workflows: dict[str, str] = WORKFLOWS,
) -> dict[str, str]:
    return keys(
        ci,
        FILTERS_TEXT,
        entries,
        read=workflows.__getitem__,
        outputs=outputs or {"client": "true", "core": "false"},
    )


def edit(path: str, blob: str) -> list[Entry]:
    return sorted([(p, b) for p, b in ENTRIES if p != path] + [(path, blob)])


class TestGlobRegex:
    """Tests translating a change map glob."""

    @pytest.mark.parametrize(
        ("glob", "path"),
        [
            ("core/**", "core/pkg/a.go"),
            ("**/*.arc", "a.arc"),
            ("**/*.arc", "x/y/a.arc"),
            ("x/go/**/*.pb.go", "x/go/a.pb.go"),
            ("x/go/**/*.pb.go", "x/go/b/a.pb.go"),
            ("uv.lock", "uv.lock"),
        ],
    )
    def test_should_match(self, glob: str, path: str) -> None:
        assert re.fullmatch(glob_regex(glob), path)

    @pytest.mark.parametrize(
        ("glob", "path"),
        [
            ("core/**", "corex/a.go"),
            ("core/*", "core/a/b.go"),
            ("uv.lock", "a/uv.lock"),
            ("x/go/**/*.pb.go", "x/go/a.go"),
        ],
    )
    def test_should_not_match(self, glob: str, path: str) -> None:
        assert not re.fullmatch(glob_regex(glob), path)

    @pytest.mark.parametrize("glob", ["!core/**", "{a,b}/**", "core/[ab].go"])
    def test_should_reject_an_unsupported_syntax(self, glob: str) -> None:
        with pytest.raises(ValueError, match="unsupported"):
            glob_regex(glob)


class TestReusable:
    """Tests choosing the jobs that can reuse a pass."""

    def test_should_key_jobs_whose_condition_reads_a_filter(self) -> None:
        jobs = load(CI)[0]["jobs"]
        assert reusable(jobs, ["client", "core"]) == {
            "client": ["client", "core"],
            "core": ["core"],
        }

    def test_should_skip_a_job_another_job_needs(self) -> None:
        jobs = load(CI)[0]["jobs"]
        assert "core-image" not in reusable(jobs, ["client", "core"])

    def test_should_skip_a_job_without_a_filter(self) -> None:
        jobs = load(CI)[0]["jobs"]
        assert "copyright" not in reusable(jobs, ["client", "core"])


class TestKeys:
    """Tests keying jobs by their inputs."""

    def test_should_be_stable(self) -> None:
        assert run() == run()
        assert set(run()) == {"client", "core"}

    def test_should_ignore_a_file_no_filter_reads(self) -> None:
        assert run(edit("docs/a.md", "d2")) == run()

    def test_should_change_the_key_of_each_job_that_reads_the_file(self) -> None:
        before, after = run(), run(edit("core/a.go", "k2"))
        assert before["core"] != after["core"]
        assert before["client"] != after["client"]

    def test_should_keep_the_key_of_a_job_that_does_not_read_the_file(self) -> None:
        before, after = run(), run(edit("client/a.ts", "c2"))
        assert before["core"] == after["core"]
        assert before["client"] != after["client"]

    def test_should_read_a_file_through_an_alias(self) -> None:
        assert run()["core"] != run(edit(".golangci.yaml", "g2"))["core"]

    def test_should_read_a_new_file(self) -> None:
        assert run()["client"] != run(edit("x/b.arc", "a1"))["client"]

    def test_should_change_with_the_ci_scripts(self) -> None:
        assert run()["core"] != run(edit(".github/scripts/a.py", "s2"))["core"]

    def test_should_change_with_a_workflow_the_job_calls(self) -> None:
        assert run()["core"] != run(edit(".github/workflows/inner.yaml", "w9"))["core"]

    def test_should_ignore_a_workflow_the_job_does_not_call(self) -> None:
        assert run()["core"] == run(edit(".github/workflows/build.yaml", "w9"))["core"]

    def test_should_change_with_the_job_block(self) -> None:
        ci = CI.replace(
            "uses: ./.github/workflows/test.yaml\n  client", "uses: x\n  client"
        )
        assert run()["core"] != run(ci=ci)["core"]

    def test_should_change_with_a_filter_value_the_inputs_read(self) -> None:
        before = run(outputs={"client": "true", "core": "false"})
        after = run(outputs={"client": "false", "core": "false"})
        assert before["client"] != after["client"]
        assert before["core"] == after["core"]


def artifact(*, expired: bool = False, repository: int = 1) -> dict[str, Any]:
    return {"expired": expired, "workflow_run": {"head_repository_id": repository}}


class TestReused:
    """Tests finding the jobs whose key passed before."""

    def check(self, artifacts: list[dict[str, Any]]) -> list[str]:
        found = {"pass-core-k": artifacts}
        return reused(
            {"client": "c", "core": "k"},
            fetch=lambda name: found.get(name, []),
            repository_id=1,
        )

    def test_should_reuse_a_live_pass_of_this_repository(self) -> None:
        assert self.check([artifact()]) == ["core"]

    def test_should_not_reuse_an_expired_pass(self) -> None:
        assert self.check([artifact(expired=True)]) == []

    def test_should_not_reuse_a_pass_of_a_fork(self) -> None:
        assert self.check([artifact(repository=2)]) == []

    def test_should_reuse_when_any_artifact_qualifies(self) -> None:
        assert self.check([artifact(repository=2), artifact()]) == ["core"]


def test_tree_should_list_every_file_with_its_blob(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    def git(*args: str) -> str:
        return subprocess.run(
            ["git", "-c", "user.name=t", "-c", "user.email=t@t", *args],
            cwd=tmp_path,
            capture_output=True,
            text=True,
            check=True,
        ).stdout.strip()

    git("init", "-q")
    (tmp_path / "b").mkdir()
    (tmp_path / "b" / "c d.go").write_text("package b\n")
    (tmp_path / "a.go").write_text("package a\n")
    git("add", ".")
    git("commit", "-qm", "init")
    monkeypatch.chdir(tmp_path)
    assert tree() == [
        ("a.go", git("rev-parse", "HEAD:a.go")),
        ("b/c d.go", git("rev-parse", "HEAD:b/c d.go")),
    ]


def test_every_keyed_job_skips_when_reused() -> None:
    root = Path(__file__).resolve().parents[3]
    jobs = load((root / WORKFLOW).read_text())[0]["jobs"]
    filters = load((root / FILTERS).read_text())[0]
    keyed = reusable(jobs, filters)
    guarded = {
        job
        for job, block in jobs.items()
        if "needs.changes.outputs.reused" in str((block or {}).get("if", ""))
    }
    assert guarded == set(keyed)
    for job in keyed:
        guard = f"!contains(fromJSON(needs.changes.outputs.reused), '{job}')"
        assert str(jobs[job]["if"]).rstrip().endswith(guard), job
