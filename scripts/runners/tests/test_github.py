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
from datetime import UTC, datetime
from email.message import Message
from http.server import BaseHTTPRequestHandler, HTTPServer

import jwt
import pytest
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa

from github import API, REPO, GitHub, send

APP_ID = "5687191"
RUNNERS = f"{API}/repos/{REPO}/actions/runners"
INSTALLATION = 180309468
NOW = datetime(2026, 11, 10, 12, tzinfo=UTC).timestamp()
HOUR = 60 * 60


@pytest.fixture(scope="module")
def key() -> rsa.RSAPrivateKey:
    return rsa.generate_private_key(public_exponent=65537, key_size=2048)


@pytest.fixture(scope="module")
def pem(key: rsa.RSAPrivateKey) -> str:
    return key.private_bytes(
        serialization.Encoding.PEM,
        serialization.PrivateFormat.PKCS8,
        serialization.NoEncryption(),
    ).decode()


class Clock:
    def __init__(self) -> None:
        self.now = NOW

    def __call__(self) -> float:
        return self.now


class Server:
    """Answers GitHub requests from memory, minting token-1, token-2, and so on, each
    valid for an hour from the clock. Fails the next request to a URL in failures with
    its status code."""

    def __init__(self, clock: Clock) -> None:
        self.clock = clock
        self.requests: list[urllib.request.Request] = []
        self.minted = 0
        self.installation = INSTALLATION
        self.failures: dict[str, int] = {}

    def __call__(self, req: urllib.request.Request) -> dict:
        self.requests.append(req)
        url = req.full_url
        if code := self.failures.pop(url, None):
            raise urllib.error.HTTPError(url, code, "failed", Message(), None)
        if url == f"{API}/repos/{REPO}/installation":
            return {"id": self.installation}
        if url == f"{API}/app/installations/{self.installation}/access_tokens":
            assert req.get_method() == "POST"
            self.minted += 1
            expires = datetime.fromtimestamp(self.clock.now + HOUR, tz=UTC)
            return {
                "token": f"token-{self.minted}",
                "expires_at": expires.strftime("%Y-%m-%dT%H:%M:%SZ"),
            }
        return {"url": url}

    def auth(self, i: int) -> str:
        return str(self.requests[i].get_header("Authorization"))


@pytest.fixture
def clock() -> Clock:
    return Clock()


@pytest.fixture
def server(clock: Clock) -> Server:
    return Server(clock)


@pytest.fixture
def github(pem: str, server: Server, clock: Clock) -> GitHub:
    return GitHub(APP_ID, pem, send=server, clock=clock)


class TestGet:
    """Tests for authenticating API reads with installation tokens."""

    def test_should_send_the_installation_token(
        self, github: GitHub, server: Server
    ) -> None:
        assert github.get("/actions/runners") == {
            "url": f"{API}/repos/{REPO}/actions/runners"
        }
        assert server.auth(-1) == "Bearer token-1"

    def test_should_mint_with_a_signed_app_jwt(
        self, github: GitHub, server: Server, key: rsa.RSAPrivateKey
    ) -> None:
        github.get("/actions/runners")
        for i in (0, 1):
            app = server.auth(i).removeprefix("Bearer ")
            claims = jwt.decode(
                app,
                key.public_key(),
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
        github.get("/actions/runners")
        clock.now += HOUR - 6 * 60
        github.get("/actions/runners")
        assert server.minted == 1
        assert server.auth(-1) == "Bearer token-1"

    def test_should_refresh_a_token_before_it_expires(
        self, github: GitHub, server: Server, clock: Clock
    ) -> None:
        github.get("/actions/runners")
        clock.now += HOUR - 4 * 60
        github.get("/actions/runners")
        assert server.minted == 2
        assert server.auth(-1) == "Bearer token-2"

    def test_should_look_up_the_installation_once(
        self, github: GitHub, server: Server, clock: Clock
    ) -> None:
        github.get("/actions/runners")
        clock.now += HOUR
        github.get("/actions/runners")
        lookups = [r for r in server.requests if r.full_url.endswith("/installation")]
        assert len(lookups) == 1

    def test_should_mint_a_new_token_after_a_rejected_one(
        self, github: GitHub, server: Server
    ) -> None:
        github.get("/actions/runners")
        server.failures[RUNNERS] = 401
        with pytest.raises(urllib.error.HTTPError):
            github.get("/actions/runners")
        github.get("/actions/runners")
        assert server.minted == 2
        assert server.auth(-1) == "Bearer token-2"

    def test_should_look_up_a_reinstalled_app_again(
        self, github: GitHub, server: Server, clock: Clock
    ) -> None:
        github.get("/actions/runners")
        stale = f"{API}/app/installations/{server.installation}/access_tokens"
        server.failures[stale] = 404
        server.installation += 1
        clock.now += HOUR
        with pytest.raises(urllib.error.HTTPError):
            github.get("/actions/runners")
        github.get("/actions/runners")
        assert server.auth(-1) == "Bearer token-2"


class Handler(BaseHTTPRequestHandler):
    def do_GET(self) -> None:
        body = json.dumps({"path": self.path}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

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
