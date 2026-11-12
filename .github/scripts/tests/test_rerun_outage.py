#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

import pytest

import rerun_outage
from rerun_outage import candidates, main, outage, sweep

REPO = "synnaxlabs/synnax"
NOW = datetime(2026, 11, 11, 21, 30, tzinfo=UTC)
SINCE = datetime(2026, 11, 11, 19, 30, tzinfo=UTC)


def run(
    id: int,
    sha: str,
    *,
    updated_at: str = "2026-11-11T21:00:48Z",
    run_attempt: int = 1,
    status: str = "completed",
    conclusion: str | None = "failure",
) -> dict[str, Any]:
    return {
        "id": id,
        "status": status,
        "conclusion": conclusion,
        "run_attempt": run_attempt,
        "head_sha": sha,
        "updated_at": updated_at,
    }


def jobs(*rows: tuple[str, str | None]) -> list[dict[str, Any]]:
    return [{"name": name, "conclusion": conclusion} for name, conclusion in rows]


# Run 43090120253 of the outage on 2026-11-11: every job passed and OK got no runner.
OK_QUEUED = run(43090120253, "b708fbf77c99b400cfe9a39236abbfa7e378b2fe")
OK_QUEUED_JOBS = jobs(
    ("changes", "success"),
    ("copyright / Check copyright headers", "success"),
    ("Fail fast", "skipped"),
    ("OK", None),
)
# Run 43094535980: jobs got no runner and OK failed on their abandoned results.
OK_FAILED = run(
    43094535980,
    "3d795c309004b3fc72acf6096dbc21ea9bae449b",
    updated_at="2026-11-11T21:02:15Z",
)
OK_FAILED_JOBS = jobs(
    ("changes", "success"),
    ("format / Check formatting", "success"),
    ("integration", "skipped"),
    ("client-ts / Test (ubuntu-latest)", None),
    ("console / Test (ubuntu-latest, 1/3)", None),
    ("arc-cpp / Lint", None),
    ("OK", "failure"),
)
# Attempt 2 of run 43089934467: a real failure.
REAL = run(43089934467, "2edc13e4a1b0c7d9e8f70615243a4b5c6d7e8f90")
REAL_JOBS = jobs(
    ("changes", "success"),
    ("docs-site / Check site", "failure"),
    ("OK", "failure"),
)


class TestOutage:
    """Tests the decision for one run and its jobs."""

    def test_should_be_true_when_ok_got_no_runner(self) -> None:
        assert outage(OK_QUEUED, OK_QUEUED_JOBS)

    def test_should_be_true_when_only_ok_failed(self) -> None:
        assert outage(OK_FAILED, OK_FAILED_JOBS)

    def test_should_be_true_when_other_jobs_were_cancelled(self) -> None:
        assert outage(OK_FAILED, [*OK_FAILED_JOBS, *jobs(("knip", "cancelled"))])

    def test_should_be_false_when_another_job_failed(self) -> None:
        assert not outage(REAL, REAL_JOBS)

    def test_should_be_false_when_another_job_timed_out(self) -> None:
        assert not outage(OK_FAILED, [*OK_FAILED_JOBS, *jobs(("knip", "timed_out"))])

    def test_should_be_false_when_a_job_with_ok_in_its_name_failed(self) -> None:
        assert not outage(OK_FAILED, [*OK_FAILED_JOBS, *jobs(("x / OK", "failure"))])

    def test_should_be_false_after_the_first_attempt(self) -> None:
        assert not outage({**OK_QUEUED, "run_attempt": 2}, OK_QUEUED_JOBS)

    @pytest.mark.parametrize(
        ("status", "conclusion"),
        [
            ("in_progress", None),
            ("queued", None),
            ("completed", "success"),
            ("completed", "cancelled"),
        ],
    )
    def test_should_be_false_for_a_run_that_did_not_fail(
        self, status: str, conclusion: str | None
    ) -> None:
        changed = {**OK_QUEUED, "status": status, "conclusion": conclusion}
        assert not outage(changed, OK_QUEUED_JOBS)


class TestCandidates:
    """Tests the choice of the runs to read the jobs of."""

    def test_should_keep_a_first_attempt_on_an_open_head(self) -> None:
        heads = {OK_QUEUED["head_sha"]}
        assert candidates([OK_QUEUED], heads, SINCE) == [OK_QUEUED]

    def test_should_drop_a_run_of_no_open_head(self) -> None:
        assert candidates([OK_QUEUED], {"other"}, SINCE) == []

    def test_should_drop_a_later_attempt(self) -> None:
        rerun = {**OK_QUEUED, "run_attempt": 2}
        assert candidates([rerun], {rerun["head_sha"]}, SINCE) == []

    def test_should_drop_a_run_that_ended_before_the_window(self) -> None:
        old = {**OK_QUEUED, "updated_at": "2026-11-11T19:29:59Z"}
        assert candidates([old], {old["head_sha"]}, SINCE) == []

    def test_should_keep_only_the_newest_run_of_a_head(self) -> None:
        older = run(1, OK_QUEUED["head_sha"])
        heads = {OK_QUEUED["head_sha"]}
        assert candidates([OK_QUEUED, older], heads, SINCE) == [OK_QUEUED]

    def test_should_drop_a_head_whose_newest_run_is_a_later_attempt(self) -> None:
        rerun = {**OK_QUEUED, "run_attempt": 2}
        older = run(1, OK_QUEUED["head_sha"])
        assert candidates([rerun, older], {rerun["head_sha"]}, SINCE) == []


class FakeAPI:
    """Answers the `gh api` calls of a sweep and records them."""

    def __init__(self, runs: list[dict[str, Any]]) -> None:
        self.runs = runs
        self.jobs: dict[int, list[dict[str, Any]]] = {}
        self.newest: dict[str, int] = {}
        self.queued: set[int] = set()
        self.heads: dict[int, str] = {}
        self.calls: list[tuple[str, ...]] = []

    def __call__(self, *args: str) -> str:
        self.calls.append(args)
        path = args[0]
        if path == "graphql":
            number = int(args[-1].removeprefix("number="))
            pull = {
                "isInMergeQueue": number in self.queued,
                "headRefOid": self.heads[number],
            }
            return json.dumps({"data": {"repository": {"pullRequest": pull}}})
        if path.endswith("/rerun"):
            return ""
        if "/jobs?" in path:
            id = int(path.split("/")[-2])
            return "".join(json.dumps(j) + "\n" for j in self.jobs[id])
        if "head_sha=" in path:
            sha = path.split("head_sha=")[1].split("&")[0]
            return f"{self.newest[sha]}\n"
        return "".join(json.dumps(r) + "\n" for r in self.runs)

    @property
    def reruns(self) -> list[str]:
        return [c[0] for c in self.calls if c[0].endswith("/rerun")]


def pulls(*runs: dict[str, Any]) -> list[dict[str, Any]]:
    return [{"number": i + 1, "headRefOid": r["head_sha"]} for i, r in enumerate(runs)]


def fake(*runs: dict[str, Any]) -> FakeAPI:
    """Returns an API where each run is the newest of its head, the head of a pull
    request outside the merge queue."""
    api = FakeAPI(list(runs))
    for pull, r in zip(pulls(*runs), runs, strict=True):
        api.newest[r["head_sha"]] = r["id"]
        api.heads[pull["number"]] = r["head_sha"]
    api.jobs = {
        OK_QUEUED["id"]: OK_QUEUED_JOBS,
        OK_FAILED["id"]: OK_FAILED_JOBS,
        REAL["id"]: REAL_JOBS,
    }
    return api


class TestSweep:
    """Tests a sweep against a fake API."""

    def test_should_rerun_only_the_outage_runs(self) -> None:
        api = fake(OK_QUEUED, REAL, OK_FAILED)
        reran = sweep(api, REPO, pulls(OK_QUEUED, REAL, OK_FAILED), NOW)
        assert reran == [OK_QUEUED["id"], OK_FAILED["id"]]
        assert api.reruns == [
            f"repos/{REPO}/actions/runs/{OK_QUEUED['id']}/rerun",
            f"repos/{REPO}/actions/runs/{OK_FAILED['id']}/rerun",
        ]

    def test_should_list_the_failed_runs_of_the_last_six_hours(self) -> None:
        api = fake()
        assert sweep(api, REPO, [], NOW) == []
        assert api.calls == [
            (
                f"repos/{REPO}/actions/workflows/ci.yaml/runs?status=failure"
                "&event=pull_request&created=>=2026-11-11T15:30:00Z&per_page=100",
                "--paginate",
                "--jq",
                ".workflow_runs[] | {id, status, conclusion, run_attempt, head_sha,"
                " updated_at} | tojson",
            )
        ]

    def test_should_page_the_jobs(self) -> None:
        api = fake(OK_QUEUED)
        sweep(api, REPO, pulls(OK_QUEUED), NOW)
        assert api.calls[1] == (
            f"repos/{REPO}/actions/runs/{OK_QUEUED['id']}/jobs?per_page=100",
            "--paginate",
            "--jq",
            ".jobs[] | {name, conclusion} | tojson",
        )

    def test_should_not_read_the_jobs_of_a_closed_pull_request(self) -> None:
        api = fake(OK_QUEUED)
        assert sweep(api, REPO, [], NOW) == []
        assert len(api.calls) == 1

    def test_should_not_rerun_a_run_that_is_not_the_newest_of_its_head(self) -> None:
        api = fake(OK_QUEUED)
        api.newest[OK_QUEUED["head_sha"]] = OK_QUEUED["id"] + 1
        assert sweep(api, REPO, pulls(OK_QUEUED), NOW) == []
        assert api.reruns == []

    def test_should_not_rerun_a_run_of_a_pull_request_in_the_merge_queue(self) -> None:
        api = fake(OK_QUEUED)
        api.queued.add(1)
        assert sweep(api, REPO, pulls(OK_QUEUED), NOW) == []
        assert api.reruns == []

    def test_should_not_rerun_a_run_of_a_head_that_moved(self) -> None:
        api = fake(OK_QUEUED)
        api.heads[1] = "moved"
        assert sweep(api, REPO, pulls(OK_QUEUED), NOW) == []
        assert api.reruns == []

    def test_should_not_rerun_when_one_pull_request_of_the_head_is_queued(self) -> None:
        api = fake(OK_QUEUED)
        api.heads[2] = OK_QUEUED["head_sha"]
        api.queued.add(2)
        both = [*pulls(OK_QUEUED), {"number": 2, "headRefOid": OK_QUEUED["head_sha"]}]
        assert sweep(api, REPO, both, NOW) == []
        assert api.reruns == []


class TestMain:
    """Tests the command line."""

    def test_should_sweep_the_pull_requests_of_the_file(
        self,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        recent = {**OK_QUEUED, "updated_at": datetime.now(UTC).isoformat()}
        api = fake(recent)
        monkeypatch.setattr(rerun_outage, "_gh_api", api)
        monkeypatch.setenv("GITHUB_REPOSITORY", REPO)
        file = tmp_path / "pulls.json"
        file.write_text(json.dumps(pulls(recent)))
        main(["rerun_outage.py", str(file)])
        assert api.reruns == [f"repos/{REPO}/actions/runs/{OK_QUEUED['id']}/rerun"]
        assert capsys.readouterr().out == "Reran CI run 43090120253 of #1\n"
