#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from pydantic import BaseModel

from freighter import Empty
from freighter.mock import MockUnaryClient


class Message(BaseModel):
    text: str


class TestMockUnaryClient:
    def test_send_returns_response(self) -> None:
        """Should return the next response and record the request."""
        client = MockUnaryClient(responses=[Message(text="hello")])
        res = client.send("", Empty(), Message)
        assert res == Message(text="hello")
        assert client.requests == [Empty()]

    def test_send_wrong_response_type(self) -> None:
        """Should raise a TypeError when the response is not of the expected type."""
        client = MockUnaryClient(responses=[Empty()])
        with pytest.raises(TypeError):
            client.send("", Empty(), Message)

    def test_send_no_responses(self) -> None:
        """Should raise a RuntimeError when no responses are left."""
        client = MockUnaryClient()
        with pytest.raises(RuntimeError):
            client.send("", Empty(), Message)

    def test_send_response_error(self) -> None:
        """Should raise the configured error."""
        client = MockUnaryClient(
            responses=[Message(text="hello")],
            response_errors=[ValueError("boom")],
        )
        with pytest.raises(ValueError, match="boom"):
            client.send("", Empty(), Message)
