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

import synnax as sy
from freighter import Empty
from freighter.mock import MockUnaryClient


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
    @pytest.fixture(scope="function")
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

    def test_retry_on_invalid_token(self, auth_setup: MockUnaryClient) -> None:
        """Test that authentication retries when receiving an invalid token error."""
        mock_client = auth_setup
        mock_client.response_errors = [sy.InvalidToken("invalid token"), None]

        response = mock_client.send("", Empty(), Empty)
        assert response == Empty()

    def test_retry_on_expired_token(self, auth_setup: MockUnaryClient) -> None:
        """Test that authentication retries when receiving an expired token error."""
        mock_client = auth_setup
        mock_client.response_errors = [sy.ExpiredToken("token expired"), None]
        response = mock_client.send("", Empty(), Empty)
        assert response == Empty()
