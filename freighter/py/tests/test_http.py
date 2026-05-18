#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from pathlib import Path

import pytest

from freighter import URL, JSONCodec, MessagePackCodec
from freighter.context import Context
from freighter.http import HTTPClient
from freighter.transport import Next

from .interface import Message


@pytest.fixture
def client(endpoint: URL) -> HTTPClient:
    http_endpoint = endpoint.child("unary")
    json_codec = JSONCodec()
    msgpack_codec = MessagePackCodec()
    return HTTPClient(http_endpoint, json_codec, [json_codec, msgpack_codec])


@pytest.mark.http
class TestSend:
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


@pytest.mark.http
class TestUpload:
    def test_json_path_negotiates_application_json(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        path = tmp_path / "in.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="hello")))
        res, err = client.upload("/echo", path, Message)
        assert err is None and res is not None
        assert res.message == "hello"
        assert res.id == 2

    def test_msgpack_path_negotiates_application_msgpack(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        path = tmp_path / "in.msgpack"
        path.write_bytes(MessagePackCodec().encode(Message(id=1, message="msg")))
        res, err = client.upload("/echo", path, Message)
        assert err is None and res is not None
        assert res.message == "msg"
        assert res.id == 2

    def test_unsupported_extension_returns_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Should reject paths whose extension has no registered codec."""
        path = tmp_path / "in.unknownext"
        path.write_bytes(b"anything")
        _, err = client.upload("/echo", path, Message)
        assert isinstance(err, ValueError)
        assert "unknownext" in str(err)

    def test_large_json_file_streams(self, client: HTTPClient, tmp_path: Path) -> None:
        big = "a" * (1024 * 1024)
        path = tmp_path / "big.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message=big)))
        res, err = client.upload("/echo", path, Message)
        assert err is None and res is not None
        assert res.message == big


@pytest.mark.http
class TestDownload:
    def test_json_dest_negotiates_application_json(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        out = tmp_path / "out.json"
        err = client.download("/echo", Message(id=1, message="hello"), out)
        assert err is None
        parsed = Message.model_validate_json(out.read_bytes())
        assert parsed.message == "hello"
        assert parsed.id == 2

    def test_msgpack_dest_negotiates_application_msgpack(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        out = tmp_path / "out.msgpack"
        err = client.download("/echo", Message(id=1, message="hi"), out)
        assert err is None
        parsed = MessagePackCodec().decode(out.read_bytes(), Message)
        assert parsed.message == "hi"
        assert parsed.id == 2

    def test_unsupported_dest_extension_returns_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Should reject destinations whose extension has no registered codec."""
        out = tmp_path / "out.unknownext"
        err = client.download("/echo", Message(id=1, message="x"), out)
        assert isinstance(err, ValueError)
        assert "unknownext" in str(err)

    def test_large_response_streams(self, client: HTTPClient, tmp_path: Path) -> None:
        big = "a" * (1024 * 1024)
        out = tmp_path / "big.json"
        err = client.download("/echo", Message(id=1, message=big), out)
        assert err is None
        parsed = Message.model_validate_json(out.read_bytes())
        assert parsed.message == big
