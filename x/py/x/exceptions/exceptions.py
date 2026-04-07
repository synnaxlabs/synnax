#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass

from pydantic import BaseModel

_TYPE_NONE = "nil"


class ExceptionPayload(BaseModel):
    """A payload that can be sent between a Freighter client and server, so that
    it can be decoded into a proper exception by the implementing language.
    """

    type: str | None
    data: str | None


def _parse_exception_payload(
    pld_or_type: ExceptionPayload | Exception | str,
    data: str | None = None,
) -> ExceptionPayload | Exception:
    """Parses the exception payload from one of the three representations:

    1. An ExceptionPayload instance. In this case, a copy of the payload is
    returned.
    2. A string encoded ExceptionPayload separated by a '---' separator.
    3. A payload type and corresponding data.

    :returns: the parsed exception payload. If the payload cannot be parsed,
    returns a payload of type unknown with as much relevant error info as possible.
    """
    if isinstance(pld_or_type, Exception):
        return pld_or_type
    elif isinstance(pld_or_type, ExceptionPayload):
        return ExceptionPayload(type=pld_or_type.type, data=pld_or_type.data)
    elif data is not None:
        return ExceptionPayload(type=pld_or_type, data=data)
    try:
        type_, data = pld_or_type.split("---", 1)
    except ValueError:
        type_, data = "unknown", pld_or_type
    return ExceptionPayload(type=type_, data=data)


EncoderFunc = Callable[[Exception], ExceptionPayload | None]
DecoderFunc = Callable[[ExceptionPayload], Exception | None]


@dataclass
class _ExceptionProvider:
    encode: EncoderFunc
    decode: DecoderFunc


class _Registry:
    providers: list[_ExceptionProvider]

    def __init__(self) -> None:
        self.providers = list()

    def register(self, provider: _ExceptionProvider) -> None:
        self.providers.append(provider)

    @staticmethod
    def encode(error: Exception | None) -> ExceptionPayload:
        raise NotImplementedError

    def decode(self, encoded: ExceptionPayload | Exception | str) -> Exception | None:
        pld = _parse_exception_payload(encoded)
        if isinstance(pld, Exception):
            return pld
        if pld.type == _TYPE_NONE:
            return None
        for provider in self.providers:
            decoded = provider.decode(pld)
            if decoded is not None:
                return decoded
        return Exception(pld.data)


REGISTRY = _Registry()


def register_exception(_encode: EncoderFunc, _decode: DecoderFunc) -> None:
    """Registers a custom error encoder and decoder with the exception registry."""
    REGISTRY.register(_ExceptionProvider(_encode, _decode))


def encode_exception(exc: Exception) -> ExceptionPayload:
    """Encodes an exception into a payload that can be sent over the network."""
    return REGISTRY.encode(exc)


def decode_exception(encoded: ExceptionPayload | None) -> Exception | None:
    """Decodes an error payload into an exception."""
    return None if encoded is None else REGISTRY.decode(encoded)


class ContiguityError(Exception):
    """Raised when time-series data is not contiguous."""

    TYPE = "sy.contiguity"
