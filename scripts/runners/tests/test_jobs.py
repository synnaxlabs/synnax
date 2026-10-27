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
from jobs import Jobs
from log import Log

LABELS = ["self-hosted", "ubuntu-test-bot"]


class Responses(GitHub):
    """Answers API paths from a table instead of the network."""

    def __init__(self, responses: dict[str, dict]) -> None:
        super().__init__("token")
        self.responses = responses

    def get(self, path: str) -> dict:
        return self.responses[path]


def runs(*ids: int) -> dict:
    return {"workflow_runs": [{"id": i} for i in ids]}


def job(id: int, status: str) -> dict:
    return {"id": id, "status": status, "labels": LABELS}


def delivery(action: str, id: int) -> dict:
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

    def __init__(self, immediate: bool = True) -> None:
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


def jobs(github: GitHub, client: Client, pool: Executor | None = None) -> Jobs:
    log = Log(core(client), Channels(core(client), False), "log")
    return Jobs(github, log, pool or Held())


class TestJobs:
    """Tests for the job queue built from deliveries and runs."""

    def test_should_count_queued_deliveries_per_pool(self, client: Client) -> None:
        j = jobs(Responses({}), client)
        j.apply(
            [
                {"action": "queued", "workflow_job": {"id": 1, "labels": LABELS}},
                {"action": "queued", "workflow_job": {"id": 2, "labels": ["x"]}},
            ]
        )
        assert j.counts() == {"ubuntu-test-bot": 1}

    def test_should_rebuild_the_queue_from_in_progress_runs(
        self, client: Client
    ) -> None:
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {
                    "jobs": [job(1, "completed"), job(2, "queued")]
                },
            }
        )
        j = jobs(github, client)
        j.resync(sy.TimeStamp.now())
        assert j.queued == {2: set(LABELS)}
        assert j.counts() == {"ubuntu-test-bot": 1}

    def test_should_keep_the_queue_until_the_resync_finishes(
        self, client: Client
    ) -> None:
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {
                    "jobs": [job(2, "queued")]
                },
            }
        )
        pool = Held(immediate=False)
        j = jobs(github, client, pool)
        j.apply([{"action": "queued", "workflow_job": {"id": 1, "labels": LABELS}}])
        j.resync(sy.TimeStamp.now())
        assert j.queued == {1: set(LABELS)}
        pool.release()
        j.resync(sy.TimeStamp.now())
        assert j.queued == {2: set(LABELS)}

    def test_should_read_every_page_of_jobs(self, client: Client) -> None:
        first = [job(i, "completed") for i in range(100)]
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {"jobs": first},
                "/actions/runs/7/jobs?per_page=100&page=2": {
                    "jobs": [job(100, "queued")]
                },
            }
        )
        j = jobs(github, client)
        j.resync(sy.TimeStamp.now())
        assert j.queued == {100: set(LABELS)}

    def test_should_restore_running_jobs_no_delivery_reported(
        self, client: Client
    ) -> None:
        running = delivery("in_progress", 3)["workflow_job"] | {"status": "in_progress"}
        github = Responses(
            {
                "/actions/runs?status=queued&per_page=100&page=1": runs(),
                "/actions/runs?status=in_progress&per_page=100&page=1": runs(7),
                "/actions/runs/7/jobs?per_page=100&page=1": {"jobs": [running]},
            }
        )
        j = jobs(github, client)
        j.resync(sy.TimeStamp.now())
        assert j.active["runner"]["id"] == 3

    def test_should_keep_the_current_job_on_a_late_completion(
        self, client: Client
    ) -> None:
        j = jobs(Responses({}), client)
        j.apply([delivery("in_progress", 2), delivery("completed", 1)])
        assert j.active["runner"]["id"] == 2
