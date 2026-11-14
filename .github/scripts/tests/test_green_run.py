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
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import pytest

SCRIPT = Path(__file__).resolve().parents[1] / "green_run.sh"
FIXTURES = Path(__file__).resolve().parent / "fixtures" / "green_run"
REPO = "synnaxlabs/synnax"

# Cut from real runs: the fields the lookup reads, for every job of every attempt.
NIGHTLY = 43152862626
RELEASE = 43139490323
RELEASE_FAILED_BUILD = 43154217723
RELEASE_RERUN = 42984679132
RELEASE_CANCELLED = 42982092558

GREEN_SHA = "6363a1466700b022a6cc2f06f72b1f6fde6c1679"
RERUN_SHA = "28ff6e2752eb3b4af19cfb703fc110fd182839cb"
CANCELLED_SHA = "6e572af7aac0cb7432be0fb647c6576131104120"

Job = dict[str, Any]
Run = dict[str, Any]

# Serves the two listings from the state file, paged as the API pages them. A fault
# whose text is in the request fails that request while its count lasts.
GH = """#!/usr/bin/env python3
import json
import os
import sys
from urllib.parse import parse_qs, urlsplit

url = urlsplit(sys.argv[2])
query = {k: v[0] for k, v in parse_qs(url.query).items()}
with open(os.environ["LOG"], "a") as log:
    log.write(sys.argv[2] + "\\n")
with open(os.environ["STATE"]) as f:
    state = json.load(f)
for fault in state["faults"]:
    if fault["match"] in sys.argv[2] and fault["count"] > 0:
        fault["count"] -= 1
        with open(os.environ["STATE"], "w") as f:
            json.dump(state, f)
        print(fault["error"], file=sys.stderr)
        sys.exit(1)
parts = url.path.split("/")
if parts[-1] == "runs":
    key = "workflow_runs"
    items = [
        r
        for r in state["runs"]
        if r["path"] == ".github/workflows/" + parts[-2]
        and (state["any_sha"] or r["head_sha"] == query["head_sha"])
    ]
else:
    key = "jobs"
    items = state["jobs"][parts[-2]]
    if query.get("filter") != "all":
        latest = max(j["run_attempt"] for j in items)
        items = [j for j in items if j["run_attempt"] == latest]
size = int(query.get("per_page", 30))
start = (int(query.get("page", 1)) - 1) * size
print(json.dumps({"total_count": len(items), key: items[start : start + size]}))
"""


def read(name: str) -> list[dict[str, Any]]:
    lines = (FIXTURES / name).read_text().splitlines()
    return [json.loads(line) for line in lines]


def with_ok(jobs: list[Job], **fields: Any) -> list[Job]:
    """Copies the jobs, with the fields of the CI job replaced."""
    return [{**job, **fields} if job["name"] == "CI / OK" else job for job in jobs]


@dataclass
class Result:
    code: int
    stdout: str
    stderr: str
    outputs: dict[str, str]
    calls: list[str]


@dataclass
class GitHub:
    """The runs and jobs the fake gh serves, and the faults it injects."""

    tmp_path: Path
    runs: list[Run] = field(default_factory=lambda: read("runs.jsonl"))
    jobs: dict[int, list[Job]] = field(
        default_factory=lambda: {
            run["id"]: read(f"jobs-{run['id']}.jsonl") for run in read("runs.jsonl")
        }
    )
    faults: list[dict[str, Any]] = field(default_factory=list)
    any_sha: bool = False

    def fail(self, match: str, error: str, count: int = 1000) -> None:
        self.faults.append({"match": match, "error": error, "count": count})

    def run(self, sha: str) -> Result:
        bin_dir = self.tmp_path / "bin"
        bin_dir.mkdir()
        gh = bin_dir / "gh"
        gh.write_text(GH)
        gh.chmod(0o755)
        state = self.tmp_path / "state.json"
        state.write_text(
            json.dumps(
                {
                    "runs": self.runs,
                    "jobs": self.jobs,
                    "faults": self.faults,
                    "any_sha": self.any_sha,
                }
            )
        )
        log = self.tmp_path / "log"
        output = self.tmp_path / "output"
        result = subprocess.run(
            [str(SCRIPT), sha],
            env={
                **os.environ,
                "PATH": f"{bin_dir}:{os.environ['PATH']}",
                "GITHUB_REPOSITORY": REPO,
                "GITHUB_OUTPUT": str(output),
                "GH_API_RETRY_DELAY": "0",
                "LOG": str(log),
                "STATE": str(state),
            },
            capture_output=True,
            text=True,
        )
        outputs = {}
        if output.exists():
            outputs = dict(
                line.split("=", 1) for line in output.read_text().splitlines()
            )
        calls = log.read_text().splitlines() if log.exists() else []
        return Result(result.returncode, result.stdout, result.stderr, outputs, calls)


@pytest.fixture
def github(tmp_path: Path) -> GitHub:
    return GitHub(tmp_path)


def assert_green(result: Result, run_id: int) -> None:
    assert result.code == 0
    assert result.stdout == f"{run_id}\n"
    assert result.outputs == {"green": "true", "run_id": str(run_id)}


def assert_not_green(result: Result) -> None:
    assert result.code == 0
    assert result.stdout == ""
    assert result.outputs == {"green": "false"}


class TestGreenRun:
    """Tests for the lookup of a run in which the full CI passed on a commit."""

    def test_should_find_a_green_nightly_run(self, github: GitHub) -> None:
        assert_green(github.run(GREEN_SHA), NIGHTLY)

    def test_should_find_a_green_release_run(self, github: GitHub) -> None:
        github.runs = [run for run in github.runs if run["id"] != NIGHTLY]
        assert_green(github.run(GREEN_SHA), RELEASE_FAILED_BUILD)

    def test_should_count_a_run_that_failed_after_ci_passed(
        self, github: GitHub
    ) -> None:
        github.runs = [run for run in github.runs if run["id"] == RELEASE_FAILED_BUILD]
        assert github.runs[0]["conclusion"] == "failure"
        assert_green(github.run(GREEN_SHA), RELEASE_FAILED_BUILD)

    def test_should_count_a_run_in_progress_after_ci_passed(
        self, github: GitHub
    ) -> None:
        github.runs = [
            {**run, "status": "in_progress", "conclusion": None}
            for run in github.runs
            if run["id"] == RELEASE
        ]
        assert_green(github.run(GREEN_SHA), RELEASE)

    def test_should_read_every_page_of_the_jobs(self, github: GitHub) -> None:
        result = github.run(GREEN_SHA)
        pages = [call for call in result.calls if f"runs/{NIGHTLY}/jobs" in call]
        assert len(github.jobs[NIGHTLY]) == 197
        assert [call.split("page=")[-1] for call in pages] == ["1", "2", "3", "4"]
        assert all("filter=all&per_page=50&" in call for call in pages)

    def test_should_read_every_page_of_the_runs(self, github: GitHub) -> None:
        red = [
            {**run, "id": 1000 + i, "path": ".github/workflows/nightly.yaml"}
            for i, run in enumerate(github.runs[:1] * 60)
        ]
        github.jobs.update(
            {run["id"]: with_ok(github.jobs[NIGHTLY][:1]) for run in red}
        )
        github.runs = [*red, *github.runs]
        result = github.run(GREEN_SHA)
        assert_green(result, NIGHTLY)
        pages = [call for call in result.calls if "nightly.yaml/runs" in call]
        assert [call.split("page=")[-1] for call in pages] == ["1", "2"]

    def test_should_not_count_a_run_in_which_ci_failed(self, github: GitHub) -> None:
        jobs = github.jobs[RELEASE_RERUN]
        github.jobs[RELEASE_RERUN] = [job for job in jobs if job["run_attempt"] == 1]
        assert_not_green(github.run(RERUN_SHA))

    def test_should_count_a_rerun_of_failed_jobs_in_which_ci_passed(
        self, github: GitHub
    ) -> None:
        assert_green(github.run(RERUN_SHA), RELEASE_RERUN)

    def test_should_not_count_a_cancelled_run(self, github: GitHub) -> None:
        assert_not_green(github.run(CANCELLED_SHA))

    @pytest.mark.parametrize(
        ("status", "conclusion"),
        [
            ("completed", "skipped"),
            ("completed", "cancelled"),
            ("completed", "failure"),
            ("completed", "timed_out"),
            ("in_progress", None),
            ("queued", None),
        ],
    )
    def test_should_not_count_a_ci_job_that_did_not_pass(
        self, github: GitHub, status: str, conclusion: str | None
    ) -> None:
        github.jobs = {
            run_id: with_ok(jobs, status=status, conclusion=conclusion)
            for run_id, jobs in github.jobs.items()
        }
        assert_not_green(github.run(GREEN_SHA))

    def test_should_not_count_a_run_without_the_ci_job(self, github: GitHub) -> None:
        github.jobs = {
            run_id: [job for job in jobs if job["name"] != "CI / OK"]
            for run_id, jobs in github.jobs.items()
        }
        assert_not_green(github.run(GREEN_SHA))

    def test_should_not_count_a_job_with_another_name(self, github: GitHub) -> None:
        github.jobs = {
            run_id: with_ok(jobs, name="OK") for run_id, jobs in github.jobs.items()
        }
        assert_not_green(github.run(GREEN_SHA))

    def test_should_find_the_earlier_attempt_of_a_run_all_jobs_rerun(
        self, github: GitHub
    ) -> None:
        github.runs = [
            {**run, "status": "in_progress", "conclusion": None, "run_attempt": 2}
            for run in github.runs
            if run["id"] == RELEASE_FAILED_BUILD
        ]
        first = github.jobs[RELEASE_FAILED_BUILD]
        second = [
            {
                "name": name,
                "status": "in_progress",
                "conclusion": None,
                "run_attempt": 2,
            }
            for name in ("Resolve versions", "Find a green CI run")
        ]
        github.jobs[RELEASE_FAILED_BUILD] = [*first, *second]
        assert_green(github.run(GREEN_SHA), RELEASE_FAILED_BUILD)

    def test_should_not_count_a_run_of_another_commit(self, github: GitHub) -> None:
        github.any_sha = True
        assert_not_green(github.run(CANCELLED_SHA))

    def test_should_not_count_a_run_of_another_workflow(self, github: GitHub) -> None:
        github.runs = [
            {**run, "path": ".github/workflows/ci.yaml"} for run in github.runs
        ]
        assert_not_green(github.run(GREEN_SHA))

    def test_should_not_count_a_run_of_another_repository(self, github: GitHub) -> None:
        github.runs = [
            {**run, "head_repository": {"full_name": "fork/synnax"}}
            for run in github.runs
        ]
        assert_not_green(github.run(GREEN_SHA))

    def test_should_retry_a_transient_api_fault(self, github: GitHub) -> None:
        github.fail(f"runs/{NIGHTLY}/jobs", "gh: Server Error (HTTP 502)", count=2)
        assert_green(github.run(GREEN_SHA), NIGHTLY)

    def test_should_go_to_the_next_run_when_the_jobs_cannot_be_read(
        self, github: GitHub
    ) -> None:
        github.fail(f"runs/{NIGHTLY}/jobs", "gh: Server Error (HTTP 502)")
        result = github.run(GREEN_SHA)
        assert_green(result, RELEASE_FAILED_BUILD)
        assert f"run {NIGHTLY}: cannot read the jobs" in result.stderr

    def test_should_go_to_the_next_workflow_when_the_runs_cannot_be_read(
        self, github: GitHub
    ) -> None:
        github.fail("nightly.yaml/runs", "gh: Not Found (HTTP 404)")
        assert_green(github.run(GREEN_SHA), RELEASE_FAILED_BUILD)

    def test_should_report_no_green_run_when_the_runs_cannot_be_read(
        self, github: GitHub
    ) -> None:
        github.fail("/runs?", "gh: Server Error (HTTP 502)")
        result = github.run(GREEN_SHA)
        assert_not_green(result)
        assert "cannot read the runs" in result.stderr

    def test_should_report_no_green_run_when_the_jobs_cannot_be_read(
        self, github: GitHub
    ) -> None:
        github.fail("/jobs?", "gh: API rate limit exceeded (HTTP 403)")
        assert_not_green(github.run(GREEN_SHA))

    def test_should_report_no_green_run_when_a_late_page_cannot_be_read(
        self, github: GitHub
    ) -> None:
        github.runs = [run for run in github.runs if run["id"] == RELEASE]
        names = [job["name"] for job in github.jobs[RELEASE]]
        assert names.index("CI / OK") < 200
        github.fail(f"runs/{RELEASE}/jobs?filter=all&per_page=50&page=5", "HTTP 403")
        assert_not_green(github.run(GREEN_SHA))

    def test_should_report_no_green_run_for_a_commit_with_no_runs(
        self, github: GitHub
    ) -> None:
        assert_not_green(github.run("0" * 40))

    def test_should_reject_a_short_sha(self, github: GitHub) -> None:
        result = github.run(GREEN_SHA[:10])
        assert result.code == 2
        assert result.outputs == {}
        assert result.calls == []
