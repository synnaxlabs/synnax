#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

from freighter import URL, encoder
from freighter.http import HTTPClientPool
from freighter.context import Context
from freighter.transport import Next

from .interface import Message


@pytest.fixture
def http_factory(endpoint: URL) -> HTTPClientPool:
    http_endpoint = endpoint.child("unary")
    return HTTPClientPool(http_endpoint, encoder.JSONEncoder())


class TestGETClient:
    @pytest.mark.focus
    def test_echo(self, http_factory: HTTPClientPool):
        """Should echo an incremented ID back to the caller."""
        res, err = http_factory.get_client().send(
            "/echo", Message(id=1, message="hello"), Message
        )
        assert err is None
        assert res.id == 2
        assert res.message == "hello"

    def test_middleware(self, http_factory: HTTPClientPool):
        dct = {"called": False}

        def mw(md: Context, next: Next) -> tuple[Context, Exception | None]:
            md.params["Test"] = "test"
            dct["called"] = True
            return next(md)

        client = http_factory.get_client()
        client.use(mw)
        res, err = client.send(
            "/middlewareCheck", Message(id=1, message="hello"), Message
        )
        assert err is None
        assert res.id == 2
        assert res.message == "hello"
        assert dct["called"]


class TestPOSTClient:
    def test_echo(self, http_factory: HTTPClientPool):
        """Should echo an incremented ID back to the caller."""
        res, err = http_factory.post_client().send(
            "/echo", Message(id=1, message="hello"), Message
        )
        assert err is None
        assert res.id == 2
        assert res.message == "hello"
