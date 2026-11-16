#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from concurrent.futures import Executor, Future
from typing import Any

import synnax as sy

from common import Channels
from conftest import Client, core
from github import GitHub
from jobs import Jobs, Queued, completed_line, ref, started_line, step_events, view
from log import Log

LABELS = ["self-hosted", "ubuntu-test-bot"]
CREATED = "2026-11-11T00:00:00Z"
WAITING = Queued(
    labels=set(LABELS), since=sy.TimeStamp(1_794_355_200 * sy.TimeSpan.SECOND)
)


def queued(id: int, *, labels: list[str] = LABELS, created: str = CREATED) -> dict:
    return {
        "action": "queued",
        "workflow_job": {"id": id, "labels": labels, "created_at": created},
    }


class Responses(GitHub):
    """Answers API paths from a table instead of the network."""

    def __init__(self, responses: dict[str, dict]) -> None:
        super().__init__("1", "")
        self.responses = responses

    def get(self, path: str) -> dict:
        return self.responses[path]


def runs(*ids: int) -> dict:
    return {"workflow_runs": [{"id": i} for i in ids]}


def job(id: int, *, status: str) -> dict:
    return {"id": id, "status": status, "labels": LABELS, "created_at": CREATED}


def delivery(action: str, *, id: int) -> dict:
    return {
        "action": action,
        "workflow_job": {
            "id": id,
            "run_id": 7,
            "name": "test",
            "html_url": "https://github.com",
            "head_branch": "main",
            "workflow_name": "CI",
            "steps": [],
            "runner_name": "runner",
            "conclusion": "success",
        },
    }


class Held(Executor):
    """Holds each submitted call until release runs it, or runs it at once when
    immediate is set."""

    def __init__(self, *, immediate: bool = True) -> None:
        self.immediate = immediate
        self.pending: list[tuple[Future[Any], Callable[[], Any]]] = []

    def submit(  # type: ignore[override]
        self, fn: Callable[[], Any], /, *args: Any, **kwargs: Any
    ) -> Future[Any]:
        future: Future[Any] = Future()
        self.pending.append((future, fn))
        if self.immediate:
            self.release()
        return future

    def release(self) -> None:
        for future, fn in self.pending:
            future.set_result(fn())
        self.pending.clear()


def jobs(github: GitHub, *, client: Client, pool: Executor | None = None) -> Jobs:
    log = Log(
        client=core(client),
        channels=Channels(client=core(client), enabled=False),
        channel="log",
    )
    return Jobs(github=github, log=log, pool=pool or Held())


class TestJobs:
    """Tests for the job queue built from deliveries and runs."""

    def test_should_count_queued_deliveries_per_pool(self, client: Client) -> None:
        j = jobs(Responses({}), client=client)
        j.apply(
            [
                queued(1),
                queued(2, labels=["x"]),
            ]
        )
        assert j.counts() == {"ubuntu-test-bot": 1}

    def test_should_count_the_jobs_of_an_unscaled_pool(self, client: Client) -> None:
        j = jobs(Responses({}), client=client)
        j.apply([queued(1, labels=["self-hosted", "macos-build-bot"])])
        assert j.counts() == {"macos-build-bot": 1}

    def test_should_report_the_wait_of_the_oldest_job_per_pool(
        self, client: Client
    ) -> None:
        j = jobs(Responses({}), client=client)
        j.apply(
            [
                queued(1, created="2026-11-11T00:00:30Z"),
                queued(2, created="2026-11-11T00:00:00Z"),
                queued(3, labels=["x"]),
            ]
        )
        now = WAITING.since + sy.TimeSpan.MINUTE
        assert j.waits(now) == {"ubuntu-test-bot": 60}

    def test_should_rebuild_the_queue_from_in_progress_runs(
        self, client: Client
    ) -> None:
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {
                    "jobs": [job(1, status="completed"), job(2, status="queued")]
                },
            }
        )
        j = jobs(github, client=client)
        j.resync(sy.TimeStamp.now())
        assert j.queued == {2: WAITING}
        assert j.counts() == {"ubuntu-test-bot": 1}

    def test_should_keep_the_queue_until_the_resync_finishes(
        self, client: Client
    ) -> None:
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {
                    "jobs": [job(2, status="queued")]
                },
            }
        )
        pool = Held(immediate=False)
        j = jobs(github, client=client, pool=pool)
        j.apply([queued(1)])
        j.resync(sy.TimeStamp.now())
        assert j.queued == {1: WAITING}
        assert not j.ready
        pool.release()
        j.resync(sy.TimeStamp.now())
        assert j.queued == {2: WAITING}
        assert j.ready

    def test_should_keep_deliveries_that_arrive_during_a_resync(
        self, client: Client
    ) -> None:
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {
                    "jobs": [job(2, status="queued")]
                },
            }
        )
        pool = Held(immediate=False)
        j = jobs(github, client=client, pool=pool)
        j.resync(sy.TimeStamp.now())
        j.apply(
            [
                queued(3),
                delivery("in_progress", id=2),
            ]
        )
        pool.release()
        j.resync(sy.TimeStamp.now())
        assert j.queued == {3: WAITING}

    def test_should_not_restore_a_job_that_finished_during_a_resync(
        self, client: Client
    ) -> None:
        running = delivery("in_progress", id=3)["workflow_job"] | {
            "status": "in_progress"
        }
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {"jobs": [running]},
            }
        )
        pool = Held(immediate=False)
        j = jobs(github, client=client, pool=pool)
        j.resync(sy.TimeStamp.now())
        pool.release()
        j.apply([delivery("in_progress", id=3), delivery("completed", id=3)])
        j.resync(sy.TimeStamp.now())
        assert "runner" not in j.active

    def test_should_read_every_page_of_jobs(self, client: Client) -> None:
        first = [job(i, status="completed") for i in range(100)]
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {"jobs": first},
                "/actions/runs/7/jobs?per_page=100&page=2": {
                    "jobs": [job(100, status="queued")]
                },
            }
        )
        j = jobs(github, client=client)
        j.resync(sy.TimeStamp.now())
        assert j.queued == {100: WAITING}

    def test_should_restore_running_jobs_no_delivery_reported(
        self, client: Client
    ) -> None:
        running = delivery("in_progress", id=3)["workflow_job"] | {
            "status": "in_progress"
        }
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {"jobs": [running]},
            }
        )
        j = jobs(github, client=client)
        j.resync(sy.TimeStamp.now())
        assert j.active["runner"]["id"] == 3

    def test_should_keep_the_current_job_on_a_late_completion(
        self, client: Client
    ) -> None:
        j = jobs(Responses({}), client=client)
        j.apply([delivery("in_progress", id=2), delivery("completed", id=1)])
        assert j.active["runner"]["id"] == 2


URL = "https://github.com/synnaxlabs/synnax/actions/runs/7/job/3"
MERGE_QUEUE = "gh-readonly-queue/main/pr-5477-097af0cf39710cc05f8e65a521bcf9acbaab1e1f"


def finished(conclusion: str, *, branch: str = "main") -> dict:
    return delivery("completed", id=3)["workflow_job"] | {
        "name": "Build (Windows)",
        "html_url": URL,
        "head_branch": branch,
        "conclusion": conclusion,
        "started_at": "2026-11-13T19:22:31Z",
        "completed_at": "2026-11-13T19:28:10Z",
    }


class TestRef:
    """Tests for the short name of the branch a job runs on."""

    def test_should_name_the_pull_request_of_a_merge_queue_branch(self) -> None:
        assert ref(MERGE_QUEUE) == "#5477"

    def test_should_name_the_ticket_of_a_ticket_branch(self) -> None:
        assert ref("sy-5823-runner-events") == "sy-5823"

    def test_should_keep_any_other_branch(self) -> None:
        assert ref("main") == "main"


class TestEventLines:
    """Tests for the text of the job and step lines."""

    def test_should_name_the_job_and_its_url_when_it_starts(self) -> None:
        line = started_line(view(finished("success", branch=MERGE_QUEUE)))
        assert line == f"CI / Build (Windows) started  #5477  {URL}"

    def test_should_leave_the_url_off_a_job_that_succeeded(self) -> None:
        line = completed_line(finished("success", branch=MERGE_QUEUE))
        assert line == "CI / Build (Windows) success 5m 39s  #5477"

    def test_should_keep_the_url_on_a_job_that_did_not_succeed(self) -> None:
        line = completed_line(finished("cancelled", branch=MERGE_QUEUE))
        assert line == f"CI / Build (Windows) cancelled 5m 39s  #5477  {URL}"

    def test_should_write_a_step_line_without_the_job(self) -> None:
        running = {
            "number": 1,
            "name": "Build Core",
            "status": "in_progress",
            "conclusion": None,
        }
        done = running | {
            "status": "completed",
            "conclusion": "success",
            "started_at": "2026-11-13T19:22:39Z",
            "completed_at": "2026-11-13T19:23:50Z",
        }
        assert step_events(before=[], after=[running]) == ["Build Core started"]
        assert step_events(before=[running], after=[done]) == [
            "Build Core success 1m 11s"
        ]
