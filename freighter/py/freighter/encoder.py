#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Protocol, Type

import msgpack
from alamos import Instrumentation, trace

from freighter.transport import P, Payload


class EncoderDecoder(Protocol):
    """Protocol for an entity that encodes and decodes values from binary."""

    def content_type(self) -> str:
        """:returns: the HTTP content type of the encoder"""
        ...

    def encode(self, data: Payload) -> bytes:
        """Encodes the given data into a binary representation.
        :param data: The data to encode.
        :returns: The binary representation of the data.
        """
        ...

    def decode(self, data: bytes, pld_t: Type[P]) -> P:
        """Decodes the given binary into a type checked payload.
        :param data: THe binary to decode.
        :param pld_t: The type of the payload to decode into.
        """
        ...


class MsgpackEncoder(EncoderDecoder):
    """A Msgpack implementation of EncoderDecoder."""

    def content_type(self):
        return "application/msgpack"

    def encode(self, payload: Payload) -> bytes:
        return msgpack.packb(payload.dict())  # type: ignore

    def decode(self, data: bytes, pld_t: Type[P]) -> P:
        return pld_t.parse_obj(msgpack.unpackb(data))


class JSONEncoder(EncoderDecoder):
    """A JSON implementation of EncoderDecoder."""

    STRING_ENCODING = "utf-8"

    def content_type(self):
        return "application/json"

    def encode(self, payload: Payload) -> bytes:
        return payload.json().encode()

    def decode(self, data: bytes, pld_t: Type[P]) -> P:
        return pld_t.parse_raw(data.decode(JSONEncoder.STRING_ENCODING))


ENCODER_DECODERS: list[EncoderDecoder] = [
    JSONEncoder(),
    MsgpackEncoder(),
]


class TracingEncoderDecoder(EncoderDecoder):
    """Injects tracing information into the context of a request."""

    wrapped: EncoderDecoder
    instrumentation: Instrumentation

    def __init__(self, wrapped: EncoderDecoder):
        self.wrapped = wrapped

    def content_type(self):
        return self.wrapped.content_type()

    @trace("debug")
    def encode(self, payload: Payload) -> bytes:
        return self.wrapped.encode(payload)

    @trace("debug")
    def decode(self, data: bytes, pld_t: Type[P]) -> P:
        return self.wrapped.decode(data, pld_t)
