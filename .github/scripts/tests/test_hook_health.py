#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io
import json
import urllib.parse
import urllib.request
from datetime import UTC, datetime, timedelta
from email.message import Message
from typing import Any

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
        healthy, _ = run(FakeAPI([delivery(502), delivery(204)]))
        assert not healthy

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

    def test_should_stop_at_the_delivery_limit(self) -> None:
        """Should read at most MAX_DELIVERIES on a busy hour."""
        api = FakeAPI([delivery(204)] * (hook_health.MAX_DELIVERIES + 250))
        _, summary = run(api)
        assert (
            summary
            == f"0 of {hook_health.MAX_DELIVERIES} deliveries in the last hour failed"
        )
        assert len(api.urls) == hook_health.MAX_DELIVERIES // 100

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

    def respond(self, body: Any, *, link: str | None) -> Http:
        headers = Message()
        if link is not None:
            headers["Link"] = link
        self.requests: list[urllib.request.Request] = []

        def urlopen(req: urllib.request.Request, timeout: float) -> Any:
            self.requests.append(req)
            return Response(json.dumps(body).encode(), headers=headers)

        return Http(token="secret", urlopen=urlopen)

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
