#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import http.client
import io
import json
import urllib.parse
import urllib.request
from datetime import UTC, datetime, timedelta
from email.message import Message
from typing import Any
from urllib.error import HTTPError, URLError

import pytest

import hook_health
from hook_health import Http, check

NOW = datetime(2026, 11, 10, 12, tzinfo=UTC)
RUN_ID = 7
REPO = "synnaxlabs/synnax"
HOOK = 42


def delivery(status: int, *, minutes_ago: float = 1) -> dict[str, Any]:
    at = NOW - timedelta(minutes=minutes_ago)
    return {
        "status_code": status,
        "delivered_at": at.isoformat().replace("+00:00", "Z"),
    }


class FakeAPI:
    """Answers deliveries and workflow run requests from fixed lists."""

    def __init__(
        self, found: list[dict[str, Any]], *, run_ids: list[int] | None = None
    ) -> None:
        self.found = found
        self.run_ids = run_ids or []
        self.urls: list[str] = []

    def __call__(self, url: str) -> tuple[Any, str | None]:
        self.urls.append(url)
        parsed = urllib.parse.urlparse(url)
        params = urllib.parse.parse_qs(parsed.query)
        if parsed.path.endswith("/actions/runs"):
            assert params["created"] == [">=2026-11-10T11:00:00Z"]
            return {"workflow_runs": [{"id": i} for i in self.run_ids]}, None
        assert parsed.path == f"/repos/{REPO}/hooks/{HOOK}/deliveries"
        page = int(params.get("page", ["1"])[0])
        batch = self.found[(page - 1) * 100 : page * 100]
        more = page * 100 < len(self.found)
        return batch, f"{url.split('&page=')[0]}&page={page + 1}" if more else None


def run(api: FakeAPI) -> tuple[bool, str]:
    return check(api, repository=REPO, hook=HOOK, run_id=RUN_ID, now=NOW)


class TestCheck:
    """Tests for judging a hook from its recent deliveries."""

    def test_should_pass_when_deliveries_succeed(self) -> None:
        """Should pass and count the deliveries when they all got a 2xx."""
        assert run(FakeAPI([delivery(204), delivery(200)])) == (
            True,
            "0 of 2 deliveries in the last hour failed",
        )

    def test_should_pass_when_fewer_than_half_failed(self) -> None:
        """Should pass through a short outage, such as a restart."""
        healthy, summary = run(FakeAPI([delivery(502), delivery(204), delivery(204)]))
        assert healthy
        assert summary == "1 of 3 deliveries in the last hour failed (1 x 502)"

    def test_should_fail_when_half_failed(self) -> None:
        """Should fail when at least half the deliveries got no 2xx."""
        healthy, _ = run(FakeAPI([delivery(502)] * 3 + [delivery(204)] * 3))
        assert not healthy

    @pytest.mark.parametrize("statuses", [[502], [502, 204], [502, 502, 204]])
    def test_should_pass_on_a_quiet_hour_with_few_failures(
        self, statuses: list[int]
    ) -> None:
        """Should need MIN_FAILURES failures before the ratio applies."""
        healthy, _ = run(FakeAPI([delivery(s) for s in statuses]))
        assert healthy

    def test_should_fail_and_count_codes_when_all_failed(self) -> None:
        """Should fail on a wrong secret and list each status code."""
        found = [delivery(401), delivery(401), delivery(0)]
        assert run(FakeAPI(found)) == (
            False,
            "3 of 3 deliveries in the last hour failed (2 x 401, 1 x 0)",
        )

    def test_should_ignore_deliveries_before_the_window(self) -> None:
        """Should judge only the last hour and stop paging at older deliveries."""
        found = [delivery(204)] * 150 + [delivery(401, minutes_ago=61)] * 300
        api = FakeAPI(found)
        assert run(api) == (True, "0 of 150 deliveries in the last hour failed")
        assert len(api.urls) == 2

    def test_should_read_the_whole_hour(self) -> None:
        """Should page through every delivery of a busy hour."""
        api = FakeAPI([delivery(204)] * 2500)
        assert run(api) == (True, "0 of 2500 deliveries in the last hour failed")
        assert len(api.urls) == 25

    def test_should_fail_when_runs_started_with_no_deliveries(self) -> None:
        """Should fail when GitHub sent nothing while other runs started."""
        assert run(FakeAPI([], run_ids=[RUN_ID, 8])) == (
            False,
            "no deliveries in the last hour while workflow runs started",
        )

    def test_should_pass_when_only_its_own_run_started(self) -> None:
        """Should pass on a quiet hour, where the check's own run is the only one."""
        healthy, _ = run(FakeAPI([], run_ids=[RUN_ID]))
        assert healthy


class Response(io.BytesIO):
    def __init__(self, body: bytes, *, headers: Message) -> None:
        super().__init__(body)
        self.headers = headers


class TestHttp:
    """Tests for the authenticated GitHub API client."""

    def respond(
        self,
        body: Any,
        *,
        link: str | None = None,
        errors: list[Exception] | None = None,
    ) -> Http:
        headers = Message()
        if link is not None:
            headers["Link"] = link
        self.requests: list[urllib.request.Request] = []
        self.sleeps: list[float] = []
        pending = list(errors or [])

        def urlopen(req: urllib.request.Request, timeout: float) -> Any:
            self.requests.append(req)
            if pending:
                raise pending.pop(0)
            return Response(json.dumps(body).encode(), headers=headers)

        return Http(token="secret", urlopen=urlopen, sleep=self.sleeps.append)

    def test_should_return_the_next_page(self) -> None:
        """Should parse the next page URL from the Link header."""
        http = self.respond(
            [1],
            link='<https://x/b>; rel="next", <https://x/a>; rel="prev"',
        )
        assert http("https://x/a") == ([1], "https://x/b")
        assert self.requests[0].get_header("Authorization") == "Bearer secret"

    @pytest.mark.parametrize("link", [None, '<https://x/a>; rel="prev"'])
    def test_should_return_no_next_page_on_the_last_page(
        self, link: str | None
    ) -> None:
        """Should return None when the Link header names no next page."""
        assert self.respond([], link=link)("https://x/a") == ([], None)

    @pytest.mark.parametrize(
        "error",
        [
            HTTPError("https://x/a", 502, "Bad Gateway", Message(), None),
            URLError("connection reset"),
            TimeoutError(),
            http.client.RemoteDisconnected("closed"),
            ConnectionResetError(),
            http.client.IncompleteRead(b"[{"),
        ],
    )
    def test_should_retry_a_transient_failure(self, error: Exception) -> None:
        """Should retry a server error or a network failure after a pause."""
        http = self.respond([1], errors=[error] * (hook_health.ATTEMPTS - 1))
        assert http("https://x/a") == ([1], None)
        assert self.sleeps == [hook_health.RETRY_SECONDS] * (hook_health.ATTEMPTS - 1)

    def test_should_raise_after_the_last_attempt(self) -> None:
        """Should raise when every attempt fails."""
        error = HTTPError("https://x/a", 503, "Unavailable", Message(), None)
        http = self.respond([1], errors=[error] * hook_health.ATTEMPTS)
        with pytest.raises(HTTPError):
            http("https://x/a")
        assert len(self.requests) == hook_health.ATTEMPTS

    def test_should_not_retry_a_client_error(self) -> None:
        """Should raise a 4xx at once, such as a token without access to hooks."""
        error = HTTPError("https://x/a", 403, "Forbidden", Message(), None)
        with pytest.raises(HTTPError):
            self.respond([1], errors=[error])("https://x/a")
        assert len(self.requests) == 1
