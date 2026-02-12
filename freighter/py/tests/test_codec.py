#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

import msgpack
from pydantic import ConfigDict, Field, BaseModel

from freighter import JSONCodec, MsgPackCodec


class PayloadWithAlias(BaseModel):
    """Payload with a field alias to test by_alias serialization.

    This is needed when the Python attribute name differs from the wire format,
    such as when using Python reserved keywords (e.g., 'from' -> 'from_').
    """

    model_config = ConfigDict(populate_by_name=True)

    from_: str = Field(alias="from")
    to: str


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


class TestMsgPackCodec:
    def test_encode_decode_basic(self) -> None:
        """Should encode and decode a basic payload."""
        codec = MsgPackCodec()
        original = PayloadWithAlias(from_="source", to="destination")
        encoded = codec.encode(original)
        decoded = codec.decode(encoded, PayloadWithAlias)
        assert decoded.from_ == "source"
        assert decoded.to == "destination"

    def test_encode_uses_alias(self) -> None:
        """Should serialize field with alias name, not Python attribute name."""

        codec = MsgPackCodec()
        payload = PayloadWithAlias(from_="source", to="destination")
        encoded = codec.encode(payload)
        data = msgpack.unpackb(encoded)
        assert isinstance(data, dict)
        # The key should be "from" (the alias), not "from_" (the Python attribute)
        assert "from" in data
        assert "from_" not in data
        assert data["from"] == "source"
