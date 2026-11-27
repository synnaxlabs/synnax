#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl

import pytest

from freighter import (
    URL,
    AsyncWebsocketClient,
    HTTPClient,
    Unreachable,
    WebsocketClient,
)
from x.codec import JSONCodec

from .interface import Message, create_registry


def _send(url: URL, ctx: ssl.SSLContext) -> Message:
    client = HTTPClient(
        url=url, codec=JSONCodec(), registry=create_registry(), ssl_context=ctx
    )
    return client.send("/echo", Message(id=1, message="hi"), Message)


def _stream(url: URL, ctx: ssl.SSLContext) -> None:
    client = WebsocketClient(
        encoder=JSONCodec(), registry=create_registry(), base_url=url, ssl_context=ctx
    )
    client.stream("/stream", Message, Message).close_send()


async def _stream_async(url: URL, ctx: ssl.SSLContext) -> None:
    client = AsyncWebsocketClient(
        encoder=JSONCodec(), registry=create_registry(), base_url=url, ssl_context=ctx
    ).with_codec(JSONCodec())
    await (await client.stream("/stream", Message, Message)).close_send()


class TestHTTP:
    """The HTTP client verifies the server's certificate and hostname."""

    def test_untrusted(self, http_url: URL) -> None:
        """Should reject a certificate that no trusted CA signs."""
        with pytest.raises(Unreachable):
            _send(http_url, ssl.create_default_context())

    def test_trusted(self, http_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should accept a certificate that the given CA signs."""
        assert _send(http_url, trusted_ctx) == Message(id=1, message="hi")

    @pytest.mark.parametrize("cert_host", ["example.com"])
    def test_hostname(self, http_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should reject a trusted certificate issued for a different host."""
        with pytest.raises(Unreachable):
            _send(http_url, trusted_ctx)


class TestWebsocket:
    """The WebSocket client verifies the server's certificate and hostname."""

    def test_untrusted(self, ws_url: URL) -> None:
        """Should reject a certificate that no trusted CA signs."""
        with pytest.raises(ssl.SSLCertVerificationError):
            _stream(ws_url, ssl.create_default_context())

    def test_trusted(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should accept a certificate that the given CA signs."""
        _stream(ws_url, trusted_ctx)

    @pytest.mark.parametrize("cert_host", ["example.com"])
    def test_hostname(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should reject a trusted certificate issued for a different host."""
        with pytest.raises(ssl.SSLCertVerificationError):
            _stream(ws_url, trusted_ctx)

    def test_session_tickets(
        self, ws_url: URL, server_ctx: ssl.SSLContext, trusted_ctx: ssl.SSLContext
    ) -> None:
        """Should open streams while the server sends TLS 1.3 session tickets."""
        server_ctx.num_tickets = 16
        client = WebsocketClient(
            encoder=JSONCodec(),
            registry=create_registry(),
            base_url=ws_url,
            ssl_context=trusted_ctx,
            open_timeout=2,
        )
        for _ in range(100):
            client.stream("/stream", Message, Message).close_send()


@pytest.mark.asyncio
class TestAsyncWebsocket:
    """The async WebSocket client verifies the server's certificate and hostname."""

    async def test_untrusted(self, ws_url: URL) -> None:
        """Should reject a certificate that no trusted CA signs."""
        with pytest.raises(ssl.SSLCertVerificationError):
            await _stream_async(ws_url, ssl.create_default_context())

    async def test_trusted(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should accept a certificate that the given CA signs."""
        await _stream_async(ws_url, trusted_ctx)

    @pytest.mark.parametrize("cert_host", ["example.com"])
    async def test_hostname(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should reject a trusted certificate issued for a different host."""
        with pytest.raises(ssl.SSLCertVerificationError):
            await _stream_async(ws_url, trusted_ctx)
