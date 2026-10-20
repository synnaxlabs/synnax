#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import json
from typing import Annotated, Protocol, TypeVar

import msgpack
from pydantic import BaseModel, BeforeValidator, PlainSerializer

P = TypeVar("P", bound=BaseModel)


def _decode_base64(value: object) -> object:
    return base64.b64decode(value) if isinstance(value, str) else value


def _encode_base64(value: bytes) -> str:
    return base64.b64encode(value).decode()


Bytes = Annotated[
    bytes,
    BeforeValidator(_decode_base64),
    PlainSerializer(_encode_base64, when_used="json"),
]
"""Binary data that crosses JSON as base64 text and MessagePack as raw bytes."""


class Codec(Protocol):
    """Protocol for an entity that encodes and decodes values from binary."""

    def content_type(self) -> str:
        """:returns: the MIME content type of the Codec"""
        ...

    def encode(self, data: BaseModel) -> bytes:
        """
        Encodes the given data into a binary representation.

        :param data: The data to encode.
        :returns: The bytes of the encoded data.
        """
        ...

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        """
        Decodes the given binary into a type checked payload.

        :param data: The binary to decode.
        :param pld_t: The Pydantic model type to decode into.
        :returns: The decoded payload.
        """
        ...


class MessagePackCodec(Codec):
    """A MessagePack implementation of Codec."""

    def content_type(self) -> str:
        return "application/msgpack"

    def encode(self, payload: BaseModel) -> bytes:
        packed = msgpack.packb(payload.model_dump(by_alias=True))
        if not isinstance(packed, bytes):
            raise ValueError(
                f"MessagePack failed to encode payload of type {type(payload).__name__}"
            )
        return packed

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        return pld_t.model_validate(msgpack.unpackb(data))


class JSONCodec(Codec):
    """A JSON implementation of Codec."""

    def content_type(self) -> str:
        return "application/json"

    def encode(self, payload: BaseModel) -> bytes:
        return payload.model_dump_json(by_alias=True, exclude_none=True).encode()

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        return pld_t.model_validate(json.loads(data.decode()))
