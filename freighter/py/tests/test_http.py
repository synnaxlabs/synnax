#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Iterator
from pathlib import Path

import pytest

from freighter import URL, JSONCodec, MessagePackCodec
from freighter.context import Context
from freighter.http import HTTPClient, stream_to_file
from freighter.transport import Next

from .interface import Message


@pytest.fixture
def client(endpoint: URL) -> HTTPClient:
    http_endpoint = endpoint.child("unary")
    json_codec = JSONCodec()
    msgpack_codec = MessagePackCodec()
    return HTTPClient(http_endpoint, json_codec, [json_codec, msgpack_codec])


@pytest.mark.http
class TestConstructor:
    def test_empty_decoders_raises_value_error(self, endpoint: URL) -> None:
        """Should reject construction when no decoders are provided."""
        with pytest.raises(ValueError, match="at least one response decoder"):
            HTTPClient(endpoint.child("unary"), JSONCodec(), [])


@pytest.mark.http
class TestSend:
    def test_echo(self, client: HTTPClient) -> None:
        """Should echo an incremented ID back to the caller."""
        res = client.send("/echo", Message(id=1, message="hello"), Message)
        assert res.id == 2
        assert res.message == "hello"

    def test_middleware(self, client: HTTPClient) -> None:
        dct = {"called": False}

        def mw(md: Context, next: Next) -> Context:
            md.params["Test"] = "test"
            dct["called"] = True
            return next(md)

        client.use(mw)
        res = client.send("/middlewareCheck", Message(id=1, message="hello"), Message)
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
        res = client.upload("/echo", path, Message)
        assert res.message == "hello"
        assert res.id == 2

    def test_msgpack_path_negotiates_application_msgpack(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        path = tmp_path / "in.msgpack"
        path.write_bytes(MessagePackCodec().encode(Message(id=1, message="msg")))
        res = client.upload("/echo", path, Message)
        assert res.message == "msg"
        assert res.id == 2

    def test_unsupported_extension_raises_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Should reject paths whose extension has no registered codec."""
        path = tmp_path / "in.unknownext"
        path.write_bytes(b"anything")
        with pytest.raises(ValueError, match="unknownext"):
            client.upload("/echo", path, Message)

    def test_large_json_file_streams(self, client: HTTPClient, tmp_path: Path) -> None:
        big = "a" * (1024 * 1024)
        path = tmp_path / "big.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message=big)))
        res = client.upload("/echo", path, Message)
        assert res.message == big


@pytest.mark.http
class TestDownload:
    def test_json_dest_negotiates_application_json(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        out = tmp_path / "out.json"
        client.download("/echo", Message(id=1, message="hello"), out)
        parsed = Message.model_validate_json(out.read_bytes())
        assert parsed.message == "hello"
        assert parsed.id == 2

    def test_msgpack_dest_negotiates_application_msgpack(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        out = tmp_path / "out.msgpack"
        client.download("/echo", Message(id=1, message="hi"), out)
        parsed = MessagePackCodec().decode(out.read_bytes(), Message)
        assert parsed.message == "hi"
        assert parsed.id == 2

    def test_unsupported_dest_extension_raises_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Should reject destinations whose extension has no registered codec."""
        out = tmp_path / "out.unknownext"
        with pytest.raises(ValueError, match="unknownext"):
            client.download("/echo", Message(id=1, message="x"), out)

    def test_large_response_streams(self, client: HTTPClient, tmp_path: Path) -> None:
        big = "a" * (1024 * 1024)
        out = tmp_path / "big.json"
        client.download("/echo", Message(id=1, message=big), out)
        parsed = Message.model_validate_json(out.read_bytes())
        assert parsed.message == big


@pytest.mark.http
class TestStreamToFile:
    def test_writes_chunks_in_order(self, tmp_path: Path) -> None:
        """Should write the concatenated chunks and leave no temp file behind."""
        dest = tmp_path / "out.json"
        stream_to_file([b'{"a":', b" 1}"], dest)
        assert dest.read_bytes() == b'{"a": 1}'
        assert list(tmp_path.glob("*.part")) == []

    def test_overwrites_existing_dest(self, tmp_path: Path) -> None:
        """Should atomically replace an existing destination file."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b'{"old": true}')
        stream_to_file([b'{"new": true}'], dest)
        assert dest.read_bytes() == b'{"new": true}'

    def test_mid_stream_failure_preserves_dest(self, tmp_path: Path) -> None:
        """A failure partway through leaves an existing dest untouched and no temp file."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b'{"old": true}')

        def chunks() -> Iterator[bytes]:
            yield b'{"partial'
            raise ConnectionError("stream broke")

        with pytest.raises(ConnectionError, match="stream broke"):
            stream_to_file(chunks(), dest)
        assert dest.read_bytes() == b'{"old": true}'
        assert list(tmp_path.glob("*.part")) == []
