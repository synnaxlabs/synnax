#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import warnings
from uuid import uuid4

import pytest

import synnax as sy
from freighter.mock import MockUnaryClient
from x.telem import TimeSpan, TimeStamp


@pytest.mark.auth
class TestClusterAuth:
    def test_invalid_credentials(self, login_info: tuple[str, int, str, str]):
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

    def test_no_password_provided(self, login_info: tuple[str, int, str, str]):
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
    def auth_setup(self) -> MockUnaryClient[int, int]:
        """Fixture that provides the mock clients and auth setup for retry tests."""
        # Setup mock login client with two successful login responses
        res = sy.auth.TokenResponse(
            token="abc",
            user=sy.User(
                key=uuid4(),
                username="synnax",
                password="seldon",
                email="synnax@synnax.com",
                first_name="Synnax",
                last_name="Labs",
            ),
        )
        mock_login_client = MockUnaryClient[
            sy.auth.InsecureCredentials, sy.auth.TokenResponse
        ](responses=[res, res], response_errors=[None, None])

        # Create auth client
        auth = sy.auth.Client(mock_login_client, "synnax", "seldon")

        # Create base mock client
        mock_client = MockUnaryClient[int, int](responses=[1, 1])
        mock_client.use(auth.middleware())

        return mock_client

    def test_retry_on_invalid_token(self, auth_setup: MockUnaryClient[int, int]):
        """Test that authentication retries when receiving an invalid token error."""
        mock_client = auth_setup
        mock_client.response_errors = [sy.InvalidToken("invalid token"), None]

        response, error = mock_client.send("", 1, int)
        assert error is None
        assert response == 1

    def test_retry_on_expired_token(self, auth_setup):
        """Test that authentication retries when receiving an expired token error."""
        mock_client = auth_setup
        mock_client.response_errors = [sy.ExpiredToken("token expired"), None]
        response, error = mock_client.send("", 1, int)
        assert error is None
        assert response == 1


def _make_token_response(node_time: TimeStamp) -> sy.auth.TokenResponse:
    return sy.auth.TokenResponse(
        token="test-token",
        user=sy.User(
            key=uuid4(),
            username="synnax",
            password="seldon",
            email="synnax@synnax.com",
            first_name="Synnax",
            last_name="Labs",
        ),
        cluster_info=sy.auth.ClusterInfo(
            cluster_key="test-cluster",
            node_version="0.54.0",
            node_key=1,
            node_time=node_time,
        ),
    )


@pytest.mark.auth
class TestClockSkewDetection:
    def test_should_measure_clock_skew(self) -> None:
        """Should measure clock skew after authentication."""
        server_time = TimeStamp.now()
        mock_client = MockUnaryClient[
            sy.auth.InsecureCredentials, sy.auth.TokenResponse
        ](responses=[_make_token_response(server_time)])
        auth = sy.auth.Client(mock_client, "synnax", "seldon")
        auth.authenticate()
        assert auth.authenticated
        assert auth.clock_skew is not None

    def test_should_warn_on_excessive_skew(self) -> None:
        """Should emit a warning when clock skew exceeds threshold."""
        far_future = TimeStamp.now() + TimeSpan.HOUR
        mock_client = MockUnaryClient[
            sy.auth.InsecureCredentials, sy.auth.TokenResponse
        ](responses=[_make_token_response(far_future)])
        auth = sy.auth.Client(
            mock_client, "synnax", "seldon", clock_skew_threshold=TimeSpan.SECOND
        )
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always")
            auth.authenticate()
            assert len(w) == 1
            assert "clock skew" in str(w[0].message)

    def test_should_not_warn_when_within_threshold(self) -> None:
        """Should not warn when clock skew is within threshold."""
        server_time = TimeStamp.now()
        mock_client = MockUnaryClient[
            sy.auth.InsecureCredentials, sy.auth.TokenResponse
        ](responses=[_make_token_response(server_time)])
        auth = sy.auth.Client(
            mock_client, "synnax", "seldon", clock_skew_threshold=TimeSpan.SECOND
        )
        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always")
            auth.authenticate()
            assert len(w) == 0

    def test_should_expose_skew_magnitude(self) -> None:
        """Should expose the measured skew via clock_skew property."""
        far_future = TimeStamp.now() + 5 * TimeSpan.SECOND
        mock_client = MockUnaryClient[
            sy.auth.InsecureCredentials, sy.auth.TokenResponse
        ](responses=[_make_token_response(far_future)])
        auth = sy.auth.Client(mock_client, "synnax", "seldon")
        with warnings.catch_warnings(record=True):
            warnings.simplefilter("always")
            auth.authenticate()
        assert int(auth.clock_skew) != 0
