#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import urllib.parse
import urllib.request
from email.message import Message
from typing import Any
from urllib.error import HTTPError

import pytest

import wait_for_turn
from wait_for_turn import GATE, INTEGRATION, GitHub, Transient


def entry(position: int, sha: str, state: str = "AWAITING_CHECKS") -> dict[str, Any]:
    return {"position": position, "state": state, "headCommit": {"oid": sha}}


class FakeGitHub(GitHub):
    """A GitHub whose HTTP responses come from a queue and a job table."""

    def __init__(self, entries: list[dict[str, Any]]) -> None:
        super().__init__("token", "synnaxlabs/synnax")
        self.entries = entries
        self.job_table: dict[str, dict[str, dict[str, Any]]] = {}
        self.failures: list[Exception] = []
        self.queue_calls = 0

    def _request(self, url: str, body: dict[str, Any] | None = None) -> Any:
        if self.failures:
            raise self.failures.pop(0)
        if body is not None:
            self.queue_calls += 1
            nodes = {"nodes": self.entries}
            return {"data": {"repository": {"mergeQueue": {"entries": nodes}}}}
        parsed = urllib.parse.urlparse(url)
        params = urllib.parse.parse_qs(parsed.query)
        if "head_sha" in params:
            sha = params["head_sha"][0]
            if sha not in self.job_table:
                return {"workflow_runs": []}
            runs = [{"id": 1, "name": "Deploy"}, {"id": ord(sha), "name": "CI"}]
            return {"workflow_runs": runs}
        run_id = int(parsed.path.split("/")[-2])
        sha = chr(run_id)
        jobs = [{"name": n, **j} for n, j in self.job_table[sha].items()]
        page = int(params["page"][0])
        return {"jobs": jobs[(page - 1) * 100 : page * 100]}

    def set_job(self, sha: str, name: str, conclusion: str | None) -> None:
        self.job_table.setdefault(sha, {})[name] = {"conclusion": conclusion}


class TestReady:
    """Tests for when a group may queue its test bot jobs."""

    def test_should_admit_a_group_within_the_slots(self) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b")])
        assert wait_for_turn.ready(github, "b", 2)

    def test_should_hold_a_group_past_the_slots(self) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b")])
        github.set_job("a", GATE, None)
        assert not wait_for_turn.ready(github, "b", 1)

    def test_should_rank_only_groups_waiting_for_checks(self) -> None:
        github = FakeGitHub(
            [entry(1, "a", "MERGEABLE"), entry(2, "b"), entry(3, "c", "QUEUED")]
        )
        assert wait_for_turn.ready(github, "b", 1)

    def test_should_admit_a_group_once_the_group_ahead_passed_its_gate(self) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b"), entry(3, "c")])
        github.set_job("b", GATE, "success")
        assert wait_for_turn.ready(github, "c", 1)

    def test_should_hold_a_group_while_the_group_ahead_waits_at_its_gate(
        self,
    ) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b")])
        github.set_job("a", INTEGRATION, None)
        github.set_job("a", GATE, None)
        assert not wait_for_turn.ready(github, "b", 0)

    def test_should_admit_a_group_behind_one_with_no_integration_suite(self) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b")])
        github.set_job("a", INTEGRATION, "skipped")
        assert wait_for_turn.ready(github, "b", 0)

    def test_should_hold_a_group_behind_one_whose_run_has_not_started(self) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b")])
        assert not wait_for_turn.ready(github, "b", 0)

    def test_should_read_every_page_of_jobs(self) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b")])
        for i in range(150):
            github.set_job("a", f"job {i}", "success")
        github.set_job("a", GATE, "success")
        assert wait_for_turn.ready(github, "b", 0)

    def test_should_exit_when_the_group_left_the_queue(self) -> None:
        github = FakeGitHub([entry(1, "a")])
        with pytest.raises(SystemExit, match="left the queue"):
            wait_for_turn.ready(github, "b", 1)


class TestWait:
    """Tests for the poll loop."""

    def test_should_poll_until_the_group_ahead_passed_its_gate(self) -> None:
        github = FakeGitHub([entry(1, "a"), entry(2, "b")])
        github.set_job("a", GATE, None)
        sleeps: list[float] = []

        def sleep(seconds: float) -> None:
            sleeps.append(seconds)
            github.set_job("a", GATE, "success")

        wait_for_turn.wait(github, "b", 1, sleep)
        assert sleeps == [wait_for_turn.POLL_SECONDS]
        assert github.queue_calls == 2

    def test_should_back_off_on_a_rate_limit(self) -> None:
        github = FakeGitHub([entry(1, "a")])
        github.failures = [Transient("x"), Transient("x")]
        sleeps: list[float] = []
        wait_for_turn.wait(github, "a", 1, sleeps.append)
        assert sleeps == [60, 120]

    def test_should_raise_an_error_that_is_not_transient(self) -> None:
        github = FakeGitHub([entry(1, "a")])
        github.failures = [HTTPError("url", 401, "Unauthorized", Message(), None)]
        with pytest.raises(HTTPError):
            wait_for_turn.wait(github, "a", 1, lambda _: None)


class TestRequest:
    """Tests for which HTTP failures count as transient."""

    @pytest.mark.parametrize(
        ("code", "header", "value"),
        [
            (403, "Retry-After", "60"),
            (403, "X-RateLimit-Remaining", "0"),
            (502, "", ""),
        ],
    )
    def test_should_treat_rate_limits_and_server_errors_as_transient(
        self, monkeypatch: pytest.MonkeyPatch, code: int, header: str, value: str
    ) -> None:
        headers = Message()
        if header:
            headers[header] = value

        def urlopen(*_: object, **__: object) -> None:
            raise HTTPError("url", code, "error", headers, None)

        monkeypatch.setattr(urllib.request, "urlopen", urlopen)
        with pytest.raises(Transient):
            GitHub("token", "synnaxlabs/synnax").queue()

    def test_should_raise_a_query_error(self, monkeypatch: pytest.MonkeyPatch) -> None:
        github = GitHub("token", "synnaxlabs/synnax")
        monkeypatch.setattr(github, "_request", lambda *_: {"errors": [{"type": "X"}]})
        with pytest.raises(RuntimeError, match="merge queue query failed"):
            github.queue()
