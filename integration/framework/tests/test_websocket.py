#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from websockets.frames import Close

from framework.websocket import is_websocket_error, suppress_websocket_errors
from freighter.websocket import ConnectionClosedError

DROPPED = ConnectionClosedError(Close(1011, "keepalive ping timeout"), None)
LOOKALIKE = RuntimeError("1011 ConnectionClosedError WebSocketException")


class TestWebSocket:
    def test_dropped_connection(self) -> None:
        """Should treat a closed connection as a WebSocket error."""
        assert is_websocket_error(DROPPED)

    def test_message_lookalike(self) -> None:
        """Should not treat an error that only mentions WebSocket terms as one."""
        assert not is_websocket_error(LOOKALIKE)

    def test_suppress_dropped_connection(self) -> None:
        """Should suppress a closed connection."""
        with suppress_websocket_errors():
            raise DROPPED

    def test_reraise_lookalike(self) -> None:
        """Should re-raise an error that only mentions WebSocket terms."""
        with pytest.raises(RuntimeError), suppress_websocket_errors():
            raise LOOKALIKE
