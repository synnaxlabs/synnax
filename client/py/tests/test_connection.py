#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

from freighter.exceptions import Unreachable
from freighter.mock import MockUnaryClient
from synnax.connection import Checker, CheckResponse, State
from x.telem import TimeSpan, TimeStamp


def _make_response(
    node_time: TimeStamp | None = None,
    node_version: str = "0.54.0",
) -> CheckResponse:
    return CheckResponse(
        cluster_key="test-cluster",
        node_version=node_version,
        node_time=node_time or TimeStamp.now(),
    )


class TestChecker:
    def test_connected_on_valid_response(self) -> None:
        """Should report connected status on a valid connectivity response."""
        mock = MockUnaryClient[None, CheckResponse](
            responses=[_make_response(), _make_response()]
        )
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
        )
        assert checker.state.status == "connected"
        assert checker.state.cluster_key == "test-cluster"
        checker.stop()

    def test_failed_on_error(self) -> None:
        """Should report failed status when the server is unreachable."""
        mock = MockUnaryClient[None, CheckResponse](
            responses=[CheckResponse()],
            response_errors=[Unreachable("localhost:9090")],
        )
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
        )
        assert checker.state.status == "failed"
        assert checker.state.error is not None
        checker.stop()

    def test_clock_skew_exceeded(self) -> None:
        """Should detect clock skew when server time is far from local time."""
        far_future = TimeStamp.now() + TimeSpan.HOUR
        mock = MockUnaryClient[None, CheckResponse](
            responses=[_make_response(node_time=far_future), _make_response()]
        )
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
            clock_skew_threshold=TimeSpan.SECOND,
        )
        assert checker.state.clock_skew_exceeded is True
        checker.stop()

    def test_clock_skew_within_threshold(self) -> None:
        """Should not flag skew when server time is close to local time."""
        mock = MockUnaryClient[None, CheckResponse](
            responses=[_make_response(), _make_response()]
        )
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
            clock_skew_threshold=TimeSpan.SECOND,
        )
        assert checker.state.clock_skew_exceeded is False
        checker.stop()

    def test_on_change_fires_on_status_transition(self) -> None:
        """Should fire onChange when status transitions."""
        responses = [
            _make_response(),
            _make_response(),
            _make_response(),
        ]
        errors: list[Exception | None] = [
            Unreachable("localhost:9090"),
            None,
            None,
        ]
        mock = MockUnaryClient[None, CheckResponse](
            responses=responses,
            response_errors=errors,
        )
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
        )
        assert checker.state.status == "failed"
        changes: list[State] = []
        checker.on_change(lambda s: changes.append(s))
        checker.check()
        assert len(changes) == 1
        assert changes[0].status == "connected"
        checker.stop()

    def test_stop_halts_background_thread(self) -> None:
        """Should stop the background polling thread."""
        mock = MockUnaryClient[None, CheckResponse](responses=[_make_response()] * 10)
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
        )
        checker.stop()
        time.sleep(0.1)
        assert not checker._thread.is_alive()

    def test_version_incompatible(self) -> None:
        """Should report incompatible versions when major.minor differs."""
        mock = MockUnaryClient[None, CheckResponse](
            responses=[_make_response(node_version="99.0.0"), _make_response()]
        )
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
        )
        assert checker.state.client_server_compatible is False
        checker.stop()

    def test_version_compatible(self) -> None:
        """Should report compatible versions when major.minor matches."""
        mock = MockUnaryClient[None, CheckResponse](
            responses=[_make_response(node_version="0.54.1"), _make_response()]
        )
        checker = Checker(
            mock,
            poll_freq=TimeSpan.SECOND * 30,
            client_version="0.54.0",
        )
        assert checker.state.client_server_compatible is True
        checker.stop()
