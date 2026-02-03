#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import pytest

from freighter import URL, codec
from freighter.context import Context
from freighter.http import HTTPClient
from freighter.transport import Next

from .interface import Message


@pytest.fixture
def client(endpoint: URL) -> HTTPClient:
    http_endpoint = endpoint.child("unary")
    return HTTPClient(http_endpoint, codec.JSONCodec())


@pytest.mark.http
class TestClient:
    def test_echo(self, client: HTTPClient) -> None:
        """Should echo an incremented ID back to the caller."""
        res, err = client.send("/echo", Message(id=1, message="hello"), Message)
        assert err is None
        assert res is not None
        assert res.id == 2
        assert res.message == "hello"

    def test_middleware(self, client: HTTPClient) -> None:
        dct = {"called": False}

        def mw(md: Context, next: Next) -> tuple[Context, Exception | None]:
            md.params["Test"] = "test"
            dct["called"] = True
            return next(md)

        client.use(mw)
        res, err = client.send(
            "/middlewareCheck", Message(id=1, message="hello"), Message
        )
        assert err is None
        assert res is not None
        assert res.id == 2
        assert res.message == "hello"
        assert dct["called"]
