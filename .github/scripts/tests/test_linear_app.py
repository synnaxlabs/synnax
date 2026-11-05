#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import io
import urllib.parse
import urllib.request

import pytest

from linear_app import linear_token

Request = tuple[str, dict[str, str]]


class TestLinearToken:
    @pytest.fixture
    def requests(self, monkeypatch: pytest.MonkeyPatch) -> list[Request]:
        sent: list[Request] = []

        def urlopen(url: str, data: bytes, timeout: float) -> io.BytesIO:
            sent.append((url, dict(urllib.parse.parse_qsl(data.decode()))))
            return io.BytesIO(b'{"access_token": "tok"}')

        monkeypatch.setattr(urllib.request, "urlopen", urlopen)
        monkeypatch.setenv("LINEAR_CLIENT_ID", "id")
        monkeypatch.setenv("LINEAR_CLIENT_SECRET", "secret")
        monkeypatch.delenv("GITHUB_ACTIONS", raising=False)
        return sent

    def test_should_request_a_client_credentials_token_and_revoke_it(
        self, requests: list[Request]
    ) -> None:
        with linear_token() as token:
            assert token == "tok"
            assert requests == [
                (
                    "https://api.linear.app/oauth/token",
                    {
                        "grant_type": "client_credentials",
                        "client_id": "id",
                        "client_secret": "secret",
                        "scope": "read,write",
                    },
                )
            ]
        assert requests[1] == (
            "https://api.linear.app/oauth/revoke",
            {"token": "tok", "token_type_hint": "access_token"},
        )

    def test_should_revoke_the_token_when_the_block_raises(
        self, requests: list[Request]
    ) -> None:
        with pytest.raises(RuntimeError), linear_token():
            raise RuntimeError
        assert requests[-1][0] == "https://api.linear.app/oauth/revoke"

    def test_should_mask_the_token_in_github_actions(
        self,
        requests: list[Request],
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        monkeypatch.setenv("GITHUB_ACTIONS", "true")
        with linear_token():
            pass
        assert capsys.readouterr().out == "::add-mask::tok\n"

    def test_should_print_nothing_outside_github_actions(
        self, requests: list[Request], capsys: pytest.CaptureFixture[str]
    ) -> None:
        with linear_token():
            pass
        assert capsys.readouterr().out == ""
