#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from typing import Protocol

import msgpack
from alamos import Instrumentation, trace

from freighter.transport import P, Payload


class Codec(Protocol):
    """Protocol for an entity that encodes and decodes values from binary."""

    def content_type(self) -> str:
        """:returns: the HTTP content type of the encoder"""
        ...

    def encode(self, data: Payload) -> bytes:
        """
        Encodes the given data into a binary representation.

        :param data: The data to encode.
        :returns: The binary representation of the data.
        """
        ...

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        """
        Decodes the given binary into a type checked payload.

        :param data: The binary to decode.
        :param pld_t: The type of the payload to decode into.
        """
        ...


class MsgPackCodec(Codec):
    """A Msgpack implementation of Codec."""

    def content_type(self) -> str:
        return "application/msgpack"

    def encode(self, payload: Payload) -> bytes:
        return msgpack.packb(payload.model_dump(by_alias=True))  # type: ignore[return-value]

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        return pld_t.model_validate(msgpack.unpackb(data))


class JSONCodec(Codec):
    """A JSON implementation of Codec."""

    STRING_ENCODING = "utf-8"

    def content_type(self) -> str:
        return "application/json"

    def encode(self, payload: Payload) -> bytes:
        return payload.model_dump_json(by_alias=True).encode()

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        return pld_t.model_validate(json.loads(data.decode(JSONCodec.STRING_ENCODING)))


CODECS: list[Codec] = [
    JSONCodec(),
    MsgPackCodec(),
]


class TracingCodec(Codec):
    """Injects tracing information into the context of a request."""

    wrapped: Codec
    instrumentation: Instrumentation

    def __init__(self, wrapped: Codec):
        self.wrapped = wrapped

    def content_type(self) -> str:
        return self.wrapped.content_type()

    @trace("debug")
    def encode(self, payload: Payload) -> bytes:
        return self.wrapped.encode(payload)

    @trace("debug")
    def decode(self, data: bytes, pld_t: type[P]) -> P:
        return self.wrapped.decode(data, pld_t)
