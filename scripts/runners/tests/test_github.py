#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import threading
import urllib.error
import urllib.request
from collections.abc import Iterator
from http.server import BaseHTTPRequestHandler, HTTPServer

import jwt
import pytest

from conftest import APP_ID, HOUR, NOW, Clock, Server, github as connect, key
from github import API, REPO, GitHub, send

RUNNERS = f"{API}/repos/{REPO}/actions/runners"
RUNNER = "ubuntu-build-bot-i-0aaa"


@pytest.fixture
def clock() -> Clock:
    return Clock()


@pytest.fixture
def server(clock: Clock) -> Server:
    return Server(clock)


@pytest.fixture
def github(server: Server) -> GitHub:
    return connect(server)


class TestCall:
    """Tests for authenticating API calls with installation tokens."""

    def test_should_send_the_installation_token(
        self, github: GitHub, server: Server
    ) -> None:
        assert github.call("/actions/runners") == {
            "url": f"{API}/repos/{REPO}/actions/runners"
        }
        assert server.auth(-1) == "Bearer token-1"

    def test_should_mint_with_a_signed_app_jwt(
        self, github: GitHub, server: Server
    ) -> None:
        github.call("/actions/runners")
        for i in (0, 1):
            app = server.auth(i).removeprefix("Bearer ")
            claims = jwt.decode(
                app,
                key().public_key(),
                algorithms=["RS256"],
                options={"verify_exp": False, "verify_iat": False},
            )
            assert claims == {
                "iss": APP_ID,
                "iat": int(NOW) - 60,
                "exp": int(NOW) + 9 * 60,
            }

    def test_should_reuse_a_token_until_near_expiry(
        self, github: GitHub, server: Server, clock: Clock
    ) -> None:
        github.call("/actions/runners")
        clock.now += HOUR - 6 * 60
        github.call("/actions/runners")
        assert server.minted == 1
        assert server.auth(-1) == "Bearer token-1"

    def test_should_refresh_a_token_before_it_expires(
        self, github: GitHub, server: Server, clock: Clock
    ) -> None:
        github.call("/actions/runners")
        clock.now += HOUR - 4 * 60
        github.call("/actions/runners")
        assert server.minted == 2
        assert server.auth(-1) == "Bearer token-2"

    def test_should_look_up_the_installation_once(
        self, github: GitHub, server: Server, clock: Clock
    ) -> None:
        github.call("/actions/runners")
        clock.now += HOUR
        github.call("/actions/runners")
        lookups = [r for r in server.requests if r.full_url.endswith("/installation")]
        assert len(lookups) == 1

    def test_should_mint_a_new_token_after_a_rejected_one(
        self, github: GitHub, server: Server
    ) -> None:
        github.call("/actions/runners")
        server.failures[RUNNERS] = 401
        with pytest.raises(urllib.error.HTTPError):
            github.call("/actions/runners")
        github.call("/actions/runners")
        assert server.minted == 2
        assert server.auth(-1) == "Bearer token-2"

    def test_should_look_up_a_reinstalled_app_again(
        self, github: GitHub, server: Server, clock: Clock
    ) -> None:
        github.call("/actions/runners")
        stale = f"{API}/app/installations/{server.installation}/access_tokens"
        server.failures[stale] = 404
        server.installation += 1
        clock.now += HOUR
        with pytest.raises(urllib.error.HTTPError):
            github.call("/actions/runners")
        github.call("/actions/runners")
        assert server.auth(-1) == "Bearer token-2"


class TestDelete:
    """Tests for removing a runner's registration."""

    def test_should_delete_the_registration(
        self, github: GitHub, server: Server
    ) -> None:
        server.registered[RUNNER] = 7
        assert github.delete(RUNNER)
        assert server.deletes() == [f"{RUNNERS}/7"]
        assert server.registered == {}

    def test_should_report_a_runner_with_a_job(
        self, github: GitHub, server: Server
    ) -> None:
        server.registered[RUNNER] = 7
        server.busy.add(RUNNER)
        assert not github.delete(RUNNER)
        assert server.registered == {RUNNER: 7}

    def test_should_succeed_without_a_registration(
        self, github: GitHub, server: Server
    ) -> None:
        assert github.delete(RUNNER)
        assert server.deletes() == []

    def test_should_raise_any_other_refusal(
        self, github: GitHub, server: Server
    ) -> None:
        server.registered[RUNNER] = 7
        server.failures[f"{RUNNERS}/7"] = 403
        with pytest.raises(urllib.error.HTTPError):
            github.delete(RUNNER)


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        body = json.dumps({"path": self.path}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_DELETE(self) -> None:
        self.send_response(204)
        self.end_headers()

    def log_message(self, format: str, *args: object) -> None:
        pass


@pytest.fixture
def local() -> Iterator[str]:
    server = HTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever)
    thread.start()
    yield f"http://127.0.0.1:{server.server_port}"
    server.shutdown()
    thread.join()
    server.server_close()


class TestSend:
    """Tests for sending a request over HTTP."""

    def test_should_decode_the_json_response(self, local: str) -> None:
        assert send(urllib.request.Request(f"{local}/x")) == {"path": "/x"}

    def test_should_return_empty_for_no_content(self, local: str) -> None:
        assert send(urllib.request.Request(f"{local}/x", method="DELETE")) == {}
