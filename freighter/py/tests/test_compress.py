#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import gzip
import zlib

import pytest

from freighter import URL
from freighter.http import HTTPClient
from x.codec import JSONCodec

from .interface import Message

LARGE_MESSAGE = "synnax telemetry payload " * 200
"""A body well above the compression floor that gzip shrinks by a wide margin."""


@pytest.mark.http
class TestRequestCompression:
    """Compressed request bodies must survive the round trip to the Go server."""

    def test_compressed_round_trip(self, endpoint: URL) -> None:
        """Should echo a gzipped request body back unchanged."""
        client = HTTPClient(endpoint.child("unary"), JSONCodec())
        res = client.send("/echo", Message(id=1, message=LARGE_MESSAGE), Message)
        assert res.message == LARGE_MESSAGE

    def test_deflate_round_trip(self, endpoint: URL) -> None:
        """Should echo a deflated request body back unchanged."""
        client = HTTPClient(endpoint.child("unary"), JSONCodec(), compression="deflate")
        res = client.send("/echo", Message(id=1, message=LARGE_MESSAGE), Message)
        assert res.message == LARGE_MESSAGE

    def test_compression_disabled(self, endpoint: URL) -> None:
        """Should round trip with request compression turned off."""
        client = HTTPClient(endpoint.child("unary"), JSONCodec(), compression=None)
        res = client.send("/echo", Message(id=1, message=LARGE_MESSAGE), Message)
        assert res.message == LARGE_MESSAGE

    def test_small_body_round_trip(self, endpoint: URL) -> None:
        """Should round trip a body below the compression floor uncompressed."""
        client = HTTPClient(endpoint.child("unary"), JSONCodec())
        res = client.send("/echo", Message(id=1, message="hello"), Message)
        assert res.message == "hello"


class TestBodySelection:
    """The client decides per body whether compressing is worth it."""

    @pytest.fixture
    def client(self) -> HTTPClient:
        return HTTPClient(URL(host="localhost", port=8080), JSONCodec())

    def test_compresses_above_the_floor(self, client: HTTPClient) -> None:
        """Should gzip a body at or above the minimum size."""
        body = b"a" * 4096
        compressed, encoding = client._maybe_compress(body)
        assert encoding == "gzip"
        assert len(compressed) < len(body)
        assert gzip.decompress(compressed) == body

    def test_skips_below_the_floor(self, client: HTTPClient) -> None:
        """Should leave a body under the minimum size alone."""
        body = b"a" * 16
        assert client._maybe_compress(body) == (body, None)

    def test_skips_when_disabled(self) -> None:
        """Should leave every body alone when compression is off."""
        client = HTTPClient(
            URL(host="localhost", port=8080), JSONCodec(), compression=None
        )
        body = b"a" * 4096
        assert client._maybe_compress(body) == (body, None)

    def test_skips_when_the_body_would_grow(self) -> None:
        """Should send a body that does not shrink under compression as-is."""
        client = HTTPClient(
            URL(host="localhost", port=8080), JSONCodec(), min_compress_size=1
        )
        # Ten bytes of gzip framing exceed anything a two-byte body could save.
        body = b"ab"
        assert client._maybe_compress(body) == (body, None)

    def test_honors_a_custom_floor(self) -> None:
        """Should compress below the default floor when the floor is lowered."""
        client = HTTPClient(
            URL(host="localhost", port=8080), JSONCodec(), min_compress_size=64
        )
        compressed, encoding = client._maybe_compress(b"a" * 128)
        assert encoding == "gzip"
        assert gzip.decompress(compressed) == b"a" * 128

    def test_deflate(self) -> None:
        """Should compress with deflate when configured to."""
        client = HTTPClient(
            URL(host="localhost", port=8080), JSONCodec(), compression="deflate"
        )
        body = b"a" * 4096
        compressed, encoding = client._maybe_compress(body)
        assert encoding == "deflate"
        assert zlib.decompress(compressed) == body
