#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid
from pathlib import Path

import pytest
from urllib3 import Retry

from freighter import URL, JSONCodec, MessagePackCodec
from freighter.context import Context
from freighter.http import HTTPClient
from freighter.transport import Next

from .interface import Error, Message


@pytest.fixture
def client(endpoint: URL) -> HTTPClient:
    return HTTPClient(endpoint.child("unary"), JSONCodec())


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
class TestFileContentTypeNegotiation:
    """The file body's Content-Type (upload) and Accept (download) are derived from the
    file path's extension at call time, independently of the codec used for the typed
    request/response. The two need not agree: the codec here is JSON, while the file
    body is MessagePack."""

    def test_upload_content_type_derived_from_extension(
        self, endpoint: URL, tmp_path: Path
    ) -> None:
        """An upload streams a MessagePack file body (Content-Type from the .msgpack
        extension) while the typed response is decoded via the JSON codec."""
        client = HTTPClient(endpoint.child("unary"), JSONCodec())
        path = tmp_path / "in.msgpack"
        path.write_bytes(MessagePackCodec().encode(Message(id=1, message="hello")))
        res = client.upload("/echo", path, Message)
        assert res.id == 2
        assert res.message == "hello"

    def test_download_accept_derived_from_extension(
        self, endpoint: URL, tmp_path: Path
    ) -> None:
        """A download requests application/msgpack (Accept from the .msgpack
        destination) while the typed request is encoded via the JSON codec."""
        client = HTTPClient(endpoint.child("unary"), JSONCodec())
        out = tmp_path / "out.msgpack"
        client.download("/echo", Message(id=1, message="hi"), out)
        parsed = MessagePackCodec().decode(out.read_bytes(), Message)
        assert parsed.message == "hi"
        assert parsed.id == 2


@pytest.mark.http
class TestRetries:
    """Exercises retry behavior against the server's /flakyUnavailable endpoint, which
    responds 503 to the first request for a given message and succeeds on every retry. A
    unary send recovers on the retry; an upload opts out of retries because it mutates
    server state and is not assumed idempotent, so the transient failure surfaces rather
    than risking a double-applied import. (The file body itself is seekable and would be
    rewound by urllib3 — the opt-out is about request semantics, not the stream.)

    The client is configured to retry on 503, the conventional transient-failure status.
    The production client (synnax.transport) retries no status codes — only connection
    errors — so a not-found or validation response is never retried.
    """

    @staticmethod
    def _retrying_client(endpoint: URL) -> HTTPClient:
        return HTTPClient(
            endpoint.child("unary"),
            JSONCodec(),
            retries=Retry(total=2, status_forcelist=[503], allowed_methods=None),
        )

    def test_send_recovers_via_retry(self, endpoint: URL) -> None:
        """A unary send replays its in-memory body and recovers from a transient 503."""
        client = self._retrying_client(endpoint)
        key = f"retry-send-{uuid.uuid4()}"
        res = client.send("/flakyUnavailable", Message(id=1, message=key), Message)
        assert res.id == 2
        assert res.message == key

    def test_upload_surfaces_first_failure(self, endpoint: URL, tmp_path: Path) -> None:
        """An upload disables retries, so the transient 503 surfaces to the caller
        rather than being retried against a non-idempotent server-side import."""
        client = self._retrying_client(endpoint)
        key = f"retry-upload-{uuid.uuid4()}"
        path = tmp_path / "in.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message=key)))
        with pytest.raises(Error):
            client.upload("/flakyUnavailable", path, Message)


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
