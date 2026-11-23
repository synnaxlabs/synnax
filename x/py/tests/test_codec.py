#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

import pytest
from pydantic import BaseModel, ConfigDict, Field, ValidationError

from x.codec import Bytes, JSONCodec


class PayloadWithAlias(BaseModel):
    """Payload with a field alias to test by_alias serialization.

    This is needed when the Python attribute name differs from the wire format,
    such as when using Python reserved keywords (e.g., 'from' -> 'from_').
    """

    model_config = ConfigDict(populate_by_name=True)

    from_: str = Field(alias="from")
    to: str


class PayloadWithOptional(BaseModel):
    """Payload with an optional field to test None omission."""

    name: str
    limit: int | None = None


class PayloadWithBoundedBytes(BaseModel):
    """Payload with a binary field of exactly three bytes."""

    data: Bytes = Field(min_length=3, max_length=3)


class PayloadWithBytes(BaseModel):
    """Payload with a binary field."""

    data: Bytes


BINARY = b"\xfb\xff\x00asm\xff"


class TestJSONCodec:
    def test_encode_decode_basic(self) -> None:
        """Should encode and decode a basic payload."""
        codec = JSONCodec()
        original = PayloadWithAlias(from_="source", to="destination")
        encoded = codec.encode(original)
        decoded = codec.decode(encoded, PayloadWithAlias)
        assert decoded.from_ == "source"
        assert decoded.to == "destination"

    def test_encode_uses_alias(self) -> None:
        """Should serialize field with alias name, not Python attribute name.

        This is critical for fields like 'from_' which must be serialized as 'from'
        to match the server's expected wire format.
        """
        codec = JSONCodec()
        payload = PayloadWithAlias(from_="source", to="destination")
        encoded = codec.encode(payload)
        data = json.loads(encoded.decode("utf-8"))
        # The key should be "from" (the alias), not "from_" (the Python attribute)
        assert "from" in data
        assert "from_" not in data
        assert data["from"] == "source"

    def test_encode_omits_none(self) -> None:
        """Should leave a None field off the wire instead of sending null."""
        codec = JSONCodec()
        encoded = codec.encode(PayloadWithOptional(name="x"))
        data = json.loads(encoded.decode("utf-8"))
        assert data == {"name": "x"}

    def test_bytes_cross_as_base64(self) -> None:
        """Should send binary data as base64 text and decode it back to bytes."""
        codec = JSONCodec()
        encoded = codec.encode(PayloadWithBytes(data=BINARY))
        assert json.loads(encoded) == {"data": "+/8AYXNt/w=="}
        assert codec.decode(encoded, PayloadWithBytes).data == BINARY

    @pytest.mark.parametrize("wire", ['"+/8AYXNt/w"', '"-_8AYXNt_w=="', "[251, 255]"])
    def test_bytes_reject_invalid_base64(self, wire: str) -> None:
        """Should reject binary data that is not standard, padded base64 text."""
        with pytest.raises(ValidationError):
            JSONCodec().decode(f'{{"data": {wire}}}'.encode(), PayloadWithBytes)

    def test_bytes_bounds_count_decoded_bytes(self) -> None:
        """Should bound binary data by its decoded bytes, not its base64 text."""
        codec = JSONCodec()
        decoded = codec.decode(b'{"data": "AQID"}', PayloadWithBoundedBytes)
        assert decoded.data == b"\x01\x02\x03"

    @pytest.mark.parametrize(
        ("text", "error"), [("AQI=", "bytes_too_short"), ("AQIDBA==", "bytes_too_long")]
    )
    def test_bytes_bounds_reject(self, text: str, error: str) -> None:
        """Should reject binary data outside its bounds, whatever its text length."""
        with pytest.raises(ValidationError) as exc:
            JSONCodec().decode(
                f'{{"data": "{text}"}}'.encode(), PayloadWithBoundedBytes
            )
        assert exc.value.errors()[0]["type"] == error
