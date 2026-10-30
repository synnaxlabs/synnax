#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import itertools
import json
import socket
import ssl
import threading
import uuid
import warnings
from pathlib import Path

import pytest
from urllib3 import Retry
from urllib3.exceptions import ProtocolError

import freighter
from freighter import URL
from freighter.context import Context
from freighter.http import HTTPClient
from freighter.transport import Next
from x.codec import JSONCodec

from .interface import Error, Message, create_registry


@pytest.fixture
def client(endpoint: URL) -> HTTPClient:
    return HTTPClient(
        url=endpoint.child("unary"), codec=JSONCodec(), registry=create_registry()
    )


class TestConstruction:
    def test_url_unchanged(self, endpoint: URL) -> None:
        """Should not change the protocol of the URL passed by the caller."""
        HTTPClient(
            url=endpoint,
            codec=JSONCodec(),
            registry=create_registry(),
            ssl_context=ssl.create_default_context(),
        )
        assert endpoint.protocol == ""

    def test_warning_filters_unchanged(self, endpoint: URL) -> None:
        """Should not change the process warning filters."""
        before = list(warnings.filters)
        HTTPClient(url=endpoint, codec=JSONCodec(), registry=create_registry())
        assert warnings.filters == before


@pytest.fixture
def unreachable_client() -> HTTPClient:
    """A client whose endpoint is a closed local port."""
    with socket.socket() as sock:
        sock.bind(("localhost", 0))
        port = sock.getsockname()[1]
    return HTTPClient(
        url=URL(host="localhost", port=port),
        codec=JSONCodec(),
        registry=create_registry(),
    )


class TestUnreachable:
    def test_send(self, unreachable_client: HTTPClient) -> None:
        """Should raise Unreachable when a send cannot connect."""
        with pytest.raises(freighter.Unreachable):
            unreachable_client.send("/echo", Message(id=1, message="hello"), Message)

    def test_upload(self, unreachable_client: HTTPClient, tmp_path: Path) -> None:
        """Should raise Unreachable when an upload cannot connect."""
        path = tmp_path / "in.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="hello")))
        with pytest.raises(freighter.Unreachable):
            unreachable_client.upload("/echo", path, Message)


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

    def test_file_name_key_always_sent(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """The file's base name accompanies every upload as the file_name key of the
        JSON-encoded params param, even when the caller supplies no params."""
        path = tmp_path / "Metrics Log.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="params")))
        res = client.upload("/paramEcho", path, Message)
        assert json.loads(res.message or "") == {"file_name": "Metrics Log.json"}

    def test_params_reach_server(self, client: HTTPClient, tmp_path: Path) -> None:
        """Caller-supplied params travel on the single params param alongside the
        automatic file_name key."""
        path = tmp_path / "Metrics Log.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="params")))
        res = client.upload(
            "/paramEcho", path, Message, params={"project": "project:abc"}
        )
        assert json.loads(res.message or "") == {
            "file_name": "Metrics Log.json",
            "project": "project:abc",
        }

    def test_caller_file_name_param_wins(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """A caller-supplied file_name key overrides the automatic one derived from
        the path."""
        path = tmp_path / "in.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="params")))
        res = client.upload(
            "/paramEcho", path, Message, params={"file_name": "Override.json"}
        )
        assert json.loads(res.message or "") == {"file_name": "Override.json"}

    def test_params_keys_never_travel_individually(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Params keys travel only inside the JSON object — never as individual
        request params."""
        path = tmp_path / "in.json"
        path.write_bytes(JSONCodec().encode(Message(id=1, message="file_name,project")))
        res = client.upload(
            "/paramEcho", path, Message, params={"project": "project:abc"}
        )
        assert res.message == "|"


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
            url=endpoint.child("unary"),
            codec=JSONCodec(),
            registry=create_registry(),
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
        client.download("/echo", Message(id=1, message="hello"), dest=out)
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
            client.download("/echo", Message(id=1, message="x"), dest=out)

    def test_unsupported_dest_extension_raises_value_error(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """Should reject destinations whose extension has no registered codec."""
        out = tmp_path / "out.unknownext"
        with pytest.raises(ValueError, match="unknownext"):
            client.download("/echo", Message(id=1, message="x"), dest=out)

    def test_empty_response_raises_and_preserves_dest(
        self, client: HTTPClient, tmp_path: Path
    ) -> None:
        """A successful but empty response is rejected rather than written as a
        zero-byte file, leaving any existing destination untouched."""
        out = tmp_path / "out.json"
        out.write_bytes(b'{"old": true}')
        with pytest.raises(ValueError, match="empty"):
            client.download("/emptyResponse", Message(id=1, message="x"), dest=out)
        assert out.read_bytes() == b'{"old": true}'

    def test_large_response_streams(self, client: HTTPClient, tmp_path: Path) -> None:
        big = "a" * (1024 * 1024)
        out = tmp_path / "big.json"
        client.download("/echo", Message(id=1, message=big), dest=out)
        parsed = Message.model_validate_json(out.read_bytes())
        assert parsed.message == big


def _send_within(client: HTTPClient, seconds: float) -> Message | None:
    """Sends a request on another thread and returns its response, or None when it
    takes longer than seconds. A request on a pool whose one connection leaked never
    completes."""
    res: list[Message] = []

    def send() -> None:
        res.append(client.send("/echo", Message(id=1, message="x"), Message))

    thread = threading.Thread(target=send, daemon=True)
    thread.start()
    thread.join(seconds)
    return res[0] if res else None


@pytest.mark.http
class TestStream:
    def test_yields_the_body(self, client: HTTPClient) -> None:
        """Should hand back the response body in chunks the caller joins."""
        with client.stream(
            "/echo", Message(id=1, message="hello"), accept="application/json"
        ) as body:
            parsed = Message.model_validate_json(b"".join(body))
        assert parsed.id == 2
        assert parsed.message == "hello"

    def test_large_body(self, client: HTTPClient) -> None:
        big = "a" * (1024 * 1024)
        with client.stream(
            "/echo", Message(id=1, message=big), accept="application/json"
        ) as body:
            assert Message.model_validate_json(b"".join(body)).message == big

    def test_server_error_raises_before_the_first_chunk(
        self, client: HTTPClient
    ) -> None:
        with (
            pytest.raises(Error),
            client.stream(
                "/middlewareCheck",
                Message(id=1, message="x"),
                accept="application/json",
            ),
        ):
            pass

    def test_encoder_failing_before_it_writes_raises(self, client: HTTPClient) -> None:
        """Should raise the error of an encoder that fails before it writes."""
        with (
            pytest.raises(Exception, match="truncating encoder failed partway"),
            client.stream(
                "/truncated",
                Message(id=0, message="ab"),
                accept="application/x-truncated",
            ),
        ):
            pass

    def test_body_cut_short_raises(self, client: HTTPClient) -> None:
        """Should raise when the server cuts the body short."""
        with (
            client.stream(
                "/truncated",
                Message(id=1 << 14, message="ab"),
                accept="application/x-truncated",
            ) as body,
            pytest.raises(ProtocolError, match="Response ended prematurely"),
        ):
            b"".join(body)

    @pytest.mark.parametrize("chunks", [0, 1, None])
    def test_leaving_releases_the_connection(
        self, endpoint: URL, chunks: int | None
    ) -> None:
        """Should release the connection when the caller leaves the context, however
        much of the body it read."""
        client = HTTPClient(
            url=endpoint.child("unary"),
            codec=JSONCodec(),
            registry=create_registry(),
            maxsize=1,
            block=True,
        )
        with client.stream(
            "/echo", Message(id=1, message="a" * (1 << 20)), accept="application/json"
        ) as body:
            for _ in itertools.islice(body, chunks):
                pass
        assert _send_within(client, 5) == Message(id=2, message="x")

    def test_middleware_failure_releases_the_connection(self, endpoint: URL) -> None:
        """Should release the connection when a middleware fails after the request."""
        client = HTTPClient(
            url=endpoint.child("unary"),
            codec=JSONCodec(),
            registry=create_registry(),
            maxsize=1,
            block=True,
        )
        failures = [RuntimeError("stop")]

        def mw(ctx: Context, next: Next) -> Context:
            out = next(ctx)
            if failures:
                raise failures.pop()
            return out

        client.use(mw)
        with (
            pytest.raises(RuntimeError, match="stop"),
            client.stream(
                "/echo", Message(id=1, message="x"), accept="application/json"
            ),
        ):
            pass
        assert _send_within(client, 5) == Message(id=2, message="x")


class TestContext:
    def test_protocol_and_target(self, endpoint: URL) -> None:
        """Should give middleware the protocol and the full request URL as target."""
        client = HTTPClient(
            url=endpoint.child("unary"), codec=JSONCodec(), registry=create_registry()
        )
        contexts: list[Context] = []

        def mw(ctx: Context, _: Next) -> Context:
            contexts.append(ctx)
            raise RuntimeError("stop")

        client.use(mw)
        with pytest.raises(RuntimeError, match="stop"):
            client.send("/echo", Message(id=1, message="hello"), Message)
        assert contexts[0].protocol == "http"
        assert contexts[0].target == "http://localhost:8080/unary/echo/"
        assert contexts[0].role == "client"
