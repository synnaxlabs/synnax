#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
import urllib.parse
from email.message import Message
from typing import Any
from urllib.error import HTTPError

import pytest

import wait_for_turn
from wait_for_turn import GATE, INTEGRATION, GitHub, Http, Transient

CONSOLE = "integration / Test (windows) / console"


def entry(position: int, *, sha: str, state: str = "AWAITING_CHECKS") -> dict[str, Any]:
    return {"position": position, "state": state, "headCommit": {"oid": sha}}


class FakeAPI:
    """Answers GitHub API requests from a merge queue and a table of CI jobs."""

    def __init__(self, entries: list[dict[str, Any]]) -> None:
        self.entries = entries
        self.job_table: dict[str, dict[str, dict[str, Any]]] = {}
        self.failures: list[Exception] = []
        self.queue_calls = 0

    def __call__(self, url: str, body: dict[str, Any] | None) -> Any:
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
        sha = chr(int(parsed.path.split("/")[-2]))
        jobs = [{"name": n, **j} for n, j in self.job_table[sha].items()]
        page = int(params["page"][0])
        return {"jobs": jobs[(page - 1) * 100 : page * 100]}

    def set_job(
        self, sha: str, *, name: str, status: str, conclusion: str | None = None
    ) -> None:
        job = {"status": status, "conclusion": conclusion}
        self.job_table.setdefault(sha, {})[name] = job

    def pass_gate(self, sha: str) -> None:
        self.set_job(sha, name=GATE, status="completed", conclusion="success")
        self.set_job(sha, name=CONSOLE, status="queued")


def github(api: FakeAPI) -> GitHub:
    return GitHub(repository="synnaxlabs/synnax", request=api)


class TestReady:
    """Tests for when a group may queue its test bot jobs."""

    def test_should_admit_a_group_within_the_slots(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        assert wait_for_turn.ready(github(api), sha="b", slots=2)

    def test_should_hold_a_group_past_the_slots(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        api.set_job("a", name=GATE, status="in_progress")
        assert not wait_for_turn.ready(github(api), sha="b", slots=1)

    def test_should_rank_only_groups_waiting_for_checks(self) -> None:
        api = FakeAPI(
            [
                entry(1, sha="a", state="MERGEABLE"),
                entry(2, sha="b"),
                entry(3, sha="c", state="QUEUED"),
            ]
        )
        assert wait_for_turn.ready(github(api), sha="b", slots=1)

    def test_should_admit_a_group_once_the_group_ahead_queued_its_jobs(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b"), entry(3, sha="c")])
        api.pass_gate("b")
        assert wait_for_turn.ready(github(api), sha="c", slots=1)

    def test_should_hold_a_group_until_the_group_ahead_queued_its_jobs(
        self,
    ) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        api.set_job("a", name=GATE, status="completed", conclusion="success")
        api.set_job("a", name=CONSOLE, status="pending")
        assert not wait_for_turn.ready(github(api), sha="b", slots=0)

    def test_should_hold_a_group_while_the_group_ahead_waits_at_its_gate(
        self,
    ) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        api.set_job("a", name=GATE, status="in_progress")
        assert not wait_for_turn.ready(github(api), sha="b", slots=0)

    def test_should_admit_a_group_behind_one_whose_gate_was_cancelled(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        api.set_job("a", name=GATE, status="completed", conclusion="cancelled")
        assert wait_for_turn.ready(github(api), sha="b", slots=0)

    def test_should_admit_a_group_behind_one_with_no_integration_suite(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        api.set_job("a", name=INTEGRATION, status="completed", conclusion="skipped")
        assert wait_for_turn.ready(github(api), sha="b", slots=0)

    def test_should_check_the_group_past_one_with_no_integration_suite(
        self,
    ) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b"), entry(3, sha="c")])
        api.set_job("a", name=GATE, status="in_progress")
        api.set_job("b", name=INTEGRATION, status="completed", conclusion="skipped")
        assert not wait_for_turn.ready(github(api), sha="c", slots=1)
        api.pass_gate("a")
        assert wait_for_turn.ready(github(api), sha="c", slots=1)

    def test_should_admit_a_group_when_no_group_ahead_runs_the_suite(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b"), entry(3, sha="c")])
        api.set_job("a", name=INTEGRATION, status="completed", conclusion="skipped")
        api.set_job("b", name=INTEGRATION, status="completed", conclusion="skipped")
        assert wait_for_turn.ready(github(api), sha="c", slots=0)

    def test_should_hold_a_group_behind_one_whose_run_has_not_started(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        assert not wait_for_turn.ready(github(api), sha="b", slots=0)

    def test_should_read_every_page_of_jobs(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        for i in range(150):
            api.set_job("a", name=f"job {i}", status="completed", conclusion="success")
        api.pass_gate("a")
        assert wait_for_turn.ready(github(api), sha="b", slots=0)

    def test_should_exit_when_the_group_left_the_queue(self) -> None:
        api = FakeAPI([entry(1, sha="a")])
        with pytest.raises(SystemExit, match="left the queue"):
            wait_for_turn.ready(github(api), sha="b", slots=1)

    def test_should_raise_a_query_error(self) -> None:
        def request(url: str, body: dict[str, Any] | None) -> Any:
            return {"errors": [{"type": "X"}]}

        with pytest.raises(RuntimeError, match="merge queue query failed"):
            GitHub(repository="synnaxlabs/synnax", request=request).queue()


class TestWait:
    """Tests for the poll loop."""

    def test_should_poll_until_the_group_ahead_queued_its_jobs(self) -> None:
        api = FakeAPI([entry(1, sha="a"), entry(2, sha="b")])
        api.set_job("a", name=GATE, status="in_progress")
        sleeps: list[float] = []

        def sleep(seconds: float) -> None:
            sleeps.append(seconds)
            api.pass_gate("a")

        wait_for_turn.wait(github(api), sha="b", slots=1, sleep=sleep)
        assert sleeps == [wait_for_turn.POLL_SECONDS]
        assert api.queue_calls == 2

    def test_should_back_off_on_a_rate_limit(self) -> None:
        api = FakeAPI([entry(1, sha="a")])
        api.failures = [Transient("x"), Transient("x")]
        sleeps: list[float] = []
        wait_for_turn.wait(github(api), sha="a", slots=1, sleep=sleeps.append)
        assert sleeps == [60, 120]

    def test_should_raise_an_error_that_is_not_transient(self) -> None:
        api = FakeAPI([entry(1, sha="a")])
        api.failures = [HTTPError("url", 401, "Unauthorized", Message(), None)]
        with pytest.raises(HTTPError):
            wait_for_turn.wait(github(api), sha="a", slots=1, sleep=lambda _: None)


class TestHttp:
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
        self, code: int, header: str, value: str
    ) -> None:
        headers = Message()
        if header:
            headers[header] = value

        def urlopen(*_: object, **__: object) -> None:
            raise HTTPError("url", code, "error", headers, None)

        with pytest.raises(Transient):
            Http(token="token", urlopen=urlopen)("https://api.github.com", None)

    def test_should_raise_a_forbidden_error_that_is_not_a_rate_limit(self) -> None:
        def urlopen(*_: object, **__: object) -> None:
            raise HTTPError("url", 403, "Forbidden", Message(), None)

        with pytest.raises(HTTPError):
            Http(token="token", urlopen=urlopen)("https://api.github.com", None)


class TestMain:
    """Tests the command line entry point."""

    @pytest.fixture
    def tokens(self, monkeypatch: pytest.MonkeyPatch) -> list[str]:
        sent: list[str] = []

        def http(*, token: str) -> FakeAPI:
            sent.append(token)
            return FakeAPI([entry(1, sha="a"), entry(2, sha="b")])

        monkeypatch.setattr(wait_for_turn, "Http", http)
        monkeypatch.setenv("GITHUB_REPOSITORY", "synnaxlabs/synnax")
        monkeypatch.setenv("GH_TOKEN", "token")
        return sent

    def test_should_return_when_the_group_is_within_the_slots(
        self, tokens: list[str], monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr(sys, "argv", ["wait_for_turn.py", "b", "2"])
        wait_for_turn.main()
        assert tokens == ["token"]

    def test_should_exit_when_the_group_is_not_in_the_queue(
        self, tokens: list[str], monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr(sys, "argv", ["wait_for_turn.py", "z", "1"])
        with pytest.raises(SystemExit, match="merge group z left the queue"):
            wait_for_turn.main()
