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

from freighter import URL
from freighter.context import Context
from freighter.http import HTTPClient
from freighter.transport import Next
from x.codec import JSONCodec

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

    def test_non_json_extension_raises_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Only JSON is supported right now, so a non-JSON extension (e.g. .msgpack) is
        rejected before any request is sent."""
        path = tmp_path / "in.msgpack"
        path.write_bytes(b"anything")
        with pytest.raises(ValueError, match="msgpack"):
            client.upload("/echo", path, Message)

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

    def test_file_name_param_always_sent(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """The file's base name accompanies every upload as the file_name param, even
        when the caller supplies no params."""
        path = tmp_path / "Metrics Log.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="file_name")))
        res = client.upload("/paramEcho", path, Message)
        assert res.message == "Metrics Log.json"

    def test_params_reach_server(self, client: HTTPClient, tmp_path: Path) -> None:
        """Caller-supplied params travel out-of-band and reach the handler alongside
        the automatic file_name param."""
        path = tmp_path / "Metrics Log.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="file_name,project")))
        res = client.upload("/paramEcho", path, Message, {"project": "project:abc"})
        assert res.message == "Metrics Log.json|project:abc"

    def test_caller_file_name_param_wins(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """A caller-supplied file_name param overrides the automatic one derived from
        the path."""
        path = tmp_path / "in.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="file_name")))
        res = client.upload("/paramEcho", path, Message, {"file_name": "Override.json"})
        assert res.message == "Override.json"

    def test_unsent_params_echo_empty(self, client: HTTPClient, tmp_path: Path) -> None:
        """A param the caller never sent reaches the handler as absent, echoed as an
        empty string."""
        path = tmp_path / "in.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="project")))
        res = client.upload("/paramEcho", path, Message)
        assert res.message == ""


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

    def test_non_json_dest_extension_raises_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Only JSON is supported right now, so a non-JSON destination (e.g. .msgpack)
        is rejected before any request is sent."""
        out = tmp_path / "out.msgpack"
        with pytest.raises(ValueError, match="msgpack"):
            client.download("/echo", Message(id=1, message="x"), out)

    def test_unsupported_dest_extension_raises_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Should reject destinations whose extension has no registered codec."""
        out = tmp_path / "out.unknownext"
        with pytest.raises(ValueError, match="unknownext"):
            client.download("/echo", Message(id=1, message="x"), out)

    def test_empty_response_raises_and_preserves_dest(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """A successful but empty response is rejected rather than written as a
        zero-byte file, leaving any existing destination untouched."""
        out = tmp_path / "out.json"
        out.write_bytes(b'{"old": true}')
        with pytest.raises(ValueError, match="empty"):
            client.download("/emptyResponse", Message(id=1, message="x"), out)
        assert out.read_bytes() == b'{"old": true}'

    def test_large_response_streams(self, client: HTTPClient, tmp_path: Path) -> None:
        big = "a" * (1024 * 1024)
        out = tmp_path / "big.json"
        client.download("/echo", Message(id=1, message=big), out)
        parsed = Message.model_validate_json(out.read_bytes())
        assert parsed.message == big
