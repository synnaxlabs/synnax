#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy
from framework.recorder import Notifications
from framework.tests.fake import Client
from synnax.status import SET_CHANNEL

WAIT = 5.0


def status(message: str) -> sy.Frame:
    raw = sy.Status(
        key="status", variant="error", message=message, time=sy.TimeStamp.now()
    ).model_dump(mode="json")
    return sy.Frame({SET_CHANNEL: sy.Series([raw])})


class TestRecorder:
    def test_wait_for_status(self) -> None:
        """Should return True once a matching status arrives."""
        client = Client()
        with Notifications(client) as notifications:
            client.stream.push(status("Started task"))
            assert notifications.wait_for("Started task", timeout=WAIT)
            notifications.check()
        assert client.stream.closed

    def test_wait_for_raises_read_error(self) -> None:
        """Should raise the error that stopped recording from a waiting call."""
        client = Client()
        with Notifications(client) as notifications:
            client.stream.push(ValueError("decode failed"))
            with pytest.raises(RuntimeError, match="decode failed") as info:
                notifications.wait_for("Started task", timeout=WAIT)
            assert isinstance(info.value.__cause__, ValueError)
            with pytest.raises(RuntimeError, match="decode failed"):
                notifications.check()
        assert client.stream.closed

    def test_check_raises_close_error(self) -> None:
        """Should raise an error from closing the stream after a stop."""
        client = Client()
        client.stream.close_error = ConnectionError("reset")
        notifications = Notifications(client)
        notifications.open()
        notifications.close()
        with pytest.raises(RuntimeError, match="reset"):
            notifications.check()

    def test_open_raises_open_error(self) -> None:
        """Should raise an error from opening the stream."""
        client = Client()
        client.open_error = ConnectionError("refused")
        with pytest.raises(RuntimeError, match="refused"):
            Notifications(client).open()
