#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import urllib.request
from datetime import UTC, datetime

import jwt
import pytest
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import rsa

from github import API, REPO, GitHub

APP_ID = "5687191"
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
    valid for an hour from the clock."""

    def __init__(self, clock: Clock) -> None:
        self.clock = clock
        self.requests: list[urllib.request.Request] = []
        self.minted = 0

    def __call__(self, req: urllib.request.Request) -> dict:
        self.requests.append(req)
        url = req.full_url
        if url == f"{API}/repos/{REPO}/installation":
            return {"id": INSTALLATION}
        if url == f"{API}/app/installations/{INSTALLATION}/access_tokens":
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
