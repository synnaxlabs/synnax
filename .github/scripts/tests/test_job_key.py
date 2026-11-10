#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import hashlib
import re
import subprocess
from pathlib import Path
from typing import Any

import pytest

from changed_jobs import FILTERS, WORKFLOW, load
from job_key import LINT, MODES, Entry, glob_regex, keys, reusable, reused, tree

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
    uses: ./.github/workflows/test.go.yaml
  client:
    needs: [changes, core-image]
    if: needs.changes.outputs.client == 'true' || needs.changes.outputs.core == 'true'
    uses: ./.github/workflows/test.ts.yaml
    with:
      tidy: ${{ needs.changes.outputs.client == 'true' }}
  exports:
    needs: changes
    if: needs.changes.outputs.core == 'true'
    runs-on: ubuntu-latest
  copyright:
    runs-on: ubuntu-latest
  ok:
    needs: [changes, client, copyright, core, core-image, exports]
    runs-on: ubuntu-latest
"""

FILTERS_TEXT = """\
go: &go
  - .golangci.yaml
core: &core
  - *go
  - core/**
client:
  - *go
  - client/**
  - "**/*.arc"
unit_tests:
  - "**/*_test.go"
  - "**/*.spec.ts"
  - "**/testdata/**"
"""

FILES = {
    ".github/scripts/a.py": "x = 1\n",
    ".github/workflows/build.yaml": "jobs: {}\n",
    ".github/workflows/inner.yaml": "jobs: {}\n",
    ".github/workflows/test.go.yaml": (
        "jobs:\n  a:\n    uses: ./.github/workflows/inner.yaml\n"
    ),
    ".github/workflows/test.ts.yaml": "jobs: {}\n",
    ".golangci.yaml": "linters: {}\n",
    "client/a.spec.ts": "it();\n",
    "client/a.ts": "export const a = 1;\n",
    "client/b.py": "a = 1  # One.\n",
    "core/a.go": "package a\n\n// A is one.\nfunc A() int { return 1 }\n",
    "core/a_test.go": "package a\n\n// Output: 1\n",
    "core/testdata/a.json": "{}\n",
    "docs/a.md": "# A\n",
}


def run_all(
    files: dict[str, str] = FILES,
    *,
    ci: str = CI,
    outputs: dict[str, str] | None = None,
) -> dict[str, dict[str, str]]:
    entries: list[Entry] = sorted(
        (p, hashlib.sha1(text.encode()).hexdigest()) for p, text in files.items()
    )
    return keys(
        ci,
        FILTERS_TEXT,
        entries,
        read=files.__getitem__,
        content=lambda p: files[p].encode(),
        outputs=outputs or {"client": "true", "core": "false"},
    )


def run(
    files: dict[str, str] = FILES,
    *,
    ci: str = CI,
    outputs: dict[str, str] | None = None,
) -> dict[str, str]:
    return {j: k["key"] for j, k in run_all(files, ci=ci, outputs=outputs).items()}


def edit(path: str, text: str) -> dict[str, str]:
    return {**FILES, path: text}


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
            "exports": ["core"],
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
        assert set(run()) == {"client", "core", "exports"}

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
            "uses: ./.github/workflows/test.go.yaml\n  client", "uses: x\n  client"
        )
        assert run()["core"] != run(ci=ci)["core"]

    @pytest.mark.parametrize("path", ["core/a_test.go", "core/testdata/a.json"])
    def test_should_skip_the_unit_tests_of_a_dependency(self, path: str) -> None:
        before, after = run(), run(edit(path, "x9"))
        assert before["client"] == after["client"]
        assert before["core"] != after["core"]

    def test_should_read_the_unit_tests_of_the_job_itself(self) -> None:
        assert run()["client"] != run(edit("client/a.spec.ts", "c4"))["client"]

    def test_should_change_with_a_filter_value_the_inputs_read(self) -> None:
        before = run(outputs={"client": "true", "core": "false"})
        after = run(outputs={"client": "false", "core": "false"})
        assert before["client"] != after["client"]
        assert before["core"] == after["core"]


GO = "package a\n\n// A is one.\nfunc A() int { return 1 }\n"


class TestComments:
    """Tests keying jobs when only comments change."""

    def comment(self) -> dict[str, dict[str, str]]:
        return run_all(edit("core/a.go", GO.replace("A is one", "A returns one")))

    def test_should_give_only_lint_jobs_a_code_key(self) -> None:
        assert {j: sorted(k) for j, k in run_all().items()} == {
            "client": ["key"],
            "core": ["code", "key"],
            "exports": ["key"],
        }

    def test_should_lint_a_lint_job_whose_own_comments_changed(self) -> None:
        before, after = run_all(), self.comment()
        assert before["core"]["key"] != after["core"]["key"]
        assert before["core"]["code"] == after["core"]["code"]

    def test_should_skip_a_code_job_whose_comments_changed(self) -> None:
        assert run_all()["client"] == self.comment()["client"]

    def test_should_skip_a_code_job_whose_own_comments_changed(self) -> None:
        after = run(edit("client/b.py", "a = 1  # Two.\n"))
        assert run()["client"] == after["client"]

    def test_should_run_a_job_with_inline_steps(self) -> None:
        assert run_all()["exports"] != self.comment()["exports"]

    def test_should_run_every_job_on_a_code_edit(self) -> None:
        before = run_all()
        after = run_all(edit("core/a.go", GO.replace("return 1", "return 2")))
        assert before["core"]["code"] != after["core"]["code"]
        assert before["client"] != after["client"]

    def test_should_count_a_directive_as_code(self) -> None:
        after = run_all(edit("core/a.go", "//go:build linux\n\n" + GO))
        assert run_all()["core"]["code"] != after["core"]["code"]

    def test_should_count_the_comments_of_a_unit_test(self) -> None:
        after = run_all(edit("core/a_test.go", "package a\n\n// Output: 2\n"))
        assert run_all()["core"]["code"] != after["core"]["code"]

    def test_should_key_a_lint_job_for_each_lint_workflow(self) -> None:
        assert {w for w, m in MODES.items() if m == LINT} == {
            "./.github/workflows/test.go.yaml",
            "./.github/workflows/test.py.yaml",
        }


def artifact(*, expired: bool = False, repository: int = 1) -> dict[str, Any]:
    return {"expired": expired, "workflow_run": {"head_repository_id": repository}}


class TestReused:
    """Tests finding the jobs whose key passed before."""

    def check(
        self, artifacts: list[dict[str, Any]], name: str = "pass-core-k"
    ) -> dict[str, list[str]]:
        found = {name: artifacts, "pass-core-c": [artifact()]}
        return reused(
            {"client": {"key": "x"}, "core": {"code": "c", "key": "k"}},
            fetch=lambda n: found.get(n, []),
            repository_id=1,
        )

    def test_should_reuse_a_live_pass_of_this_repository(self) -> None:
        assert self.check([artifact()]) == {"lint_only": [], "reused": ["core"]}

    def test_should_not_reuse_an_expired_pass(self) -> None:
        assert self.check([artifact(expired=True)])["reused"] == []

    def test_should_not_reuse_a_pass_of_a_fork(self) -> None:
        assert self.check([artifact(repository=2)])["reused"] == []

    def test_should_reuse_when_any_artifact_qualifies(self) -> None:
        assert self.check([artifact(repository=2), artifact()])["reused"] == ["core"]

    def test_should_lint_only_when_only_the_code_passed(self) -> None:
        assert self.check([]) == {"lint_only": ["core"], "reused": []}


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


ROOT = Path(__file__).resolve().parents[3]


def real() -> tuple[dict[str, Any], dict[str, list[str]]]:
    jobs = load((ROOT / WORKFLOW).read_text())[0]["jobs"]
    return jobs, reusable(jobs, load((ROOT / FILTERS).read_text())[0])


def test_each_job_reads_its_own_filter_first() -> None:
    deps = {job: refs[1:] for job, refs in real()[1].items() if len(refs) > 1}
    core = ["core"]
    assert deps == {
        "arc-cpp": core,
        "client-cpp": core,
        "client-py": core,
        "client-py-lowest": core,
        "client-ts": core,
        "console": core,
        "driver": core,
        "freighter-py": ["freighter_integration"],
        "freighter-py-lowest": ["freighter_integration"],
        "freighter-ts": ["freighter_integration"],
        "integration": ["console_build", "core_build", "driver_build"],
        "pluto": core,
    }


def test_every_keyed_job_skips_when_reused() -> None:
    jobs, keyed = real()
    guarded = {
        job
        for job, block in jobs.items()
        if "needs.changes.outputs.reused" in str((block or {}).get("if", ""))
    }
    assert guarded == set(keyed)
    for job in keyed:
        guard = f"!contains(fromJSON(needs.changes.outputs.reused), '{job}')"
        assert str(jobs[job]["if"]).rstrip().endswith(guard), job


def test_every_lint_job_runs_only_its_lint_when_its_code_passed() -> None:
    jobs, keyed = real()
    lint = {j for j in keyed if MODES.get(jobs[j].get("uses", "")) == LINT}
    reads = {
        j for j, block in jobs.items() if j != "changes" and "lint_only" in str(block)
    }
    assert reads == lint
    for job in lint:
        lint_only = f"!contains(fromJSON(needs.changes.outputs.lint_only), '{job}')"
        test = f"${{{{ inputs.full || {lint_only} }}}}"
        assert " ".join(jobs[job]["with"]["test"].split()) == test, job
