#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4

import pytest

import freighter
import synnax as sy
from freighter import Empty
from freighter.mock import MockUnaryClient
from synnax.exceptions import create_registry


@pytest.mark.auth
class TestClusterAuth:
    def test_invalid_credentials(self, login_info: tuple[str, int, str, str]) -> None:
        """
        Should raise an AuthError
        """
        host, port, username, _ = login_info
        with pytest.raises(sy.AuthError):
            sy.Synnax(
                host=host,
                port=port,
                username=username,
                password="wrong",
            )

    def test_no_password_provided(self, login_info: tuple[str, int, str, str]) -> None:
        """
        Should raise a ValidationError
        """
        host, port, username, _ = login_info
        with pytest.raises(sy.PathError):
            sy.Synnax(
                host=host,
                port=port,
                username=username,
            )


@pytest.mark.auth
class TestAuthRetry:
    @pytest.fixture
    def auth_setup(self) -> MockUnaryClient:
        """Fixture that provides the mock clients and auth setup for retry tests."""
        # Setup mock login client with two successful login responses
        res = sy.auth.TokenResponse(
            token="abc",
            user=sy.User(
                key=uuid4(),
                username="synnax",
                first_name="Synnax",
                last_name="Labs",
            ),
        )
        mock_login_client = MockUnaryClient(
            responses=[res, res], response_errors=[None, None]
        )

        # Create auth client
        auth = sy.auth.Client(mock_login_client, "synnax", "seldon")

        # Create base mock client
        mock_client = MockUnaryClient(responses=[Empty(), Empty()])
        mock_client.use(auth.middleware())

        return mock_client

    @pytest.mark.parametrize(
        ("type_", "exc"),
        [
            ("sy.auth", sy.AuthError),
            ("sy.auth.invalid-credentials", sy.exceptions.InvalidCredentials),
            ("sy.auth.invalid_token", sy.InvalidToken),
            ("sy.auth.expired_token", sy.ExpiredToken),
            ("sy.auth.access_denied", sy.AccessDenied),
            ("sy.auth.repeated-username", sy.AuthError),
        ],
    )
    def test_decode(self, type_: str, exc: type[sy.AuthError]) -> None:
        """Should decode each auth error type the Core sends."""
        err = create_registry().decode(
            freighter.ExceptionPayload(type=type_, data="refused")
        )
        assert type(err) is exc
        assert str(err) == "refused"

    @pytest.mark.parametrize(
        "type_", ["sy.auth.invalid_token", "sy.auth.expired_token"]
    )
    def test_retry_on_decoded_token_error(
        self, auth_setup: MockUnaryClient, type_: str
    ) -> None:
        """Should re-authenticate and retry after a token error from the Core."""
        mock_client = auth_setup
        mock_client.response_errors = [
            create_registry().decode(
                freighter.ExceptionPayload(type=type_, data="refused")
            ),
            None,
        ]
        assert mock_client.send("", Empty(), Empty) == Empty()
