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
_TYPE_UNKNOWN = "unknown"


class ExceptionPayload(BaseModel):
    """A payload that can be sent between a Freighter client and server, so that
    it can be decoded into a proper exception by the implementing language.
    """

    type: str | None
    data: str | None


EncoderFunc = Callable[[Exception], ExceptionPayload | None]
"""Encodes an exception, or returns None when it cannot handle the exception."""
DecoderFunc = Callable[[ExceptionPayload], Exception | None]
"""Decodes a payload, or returns None when it cannot handle the payload's type."""


@dataclass(frozen=True)
class Provider:
    """Encodes and decodes the exceptions of one category."""

    encode: EncoderFunc
    decode: DecoderFunc


class Registry:
    """Encodes and decodes exceptions through the providers registered with it."""

    _providers: list[Provider]

    def __init__(self) -> None:
        self._providers = []

    def register(self, provider: Provider) -> None:
        """Adds a provider. Encode and decode try providers in registration order."""
        self._providers.append(provider)

    def encode(self, exc: Exception | None) -> ExceptionPayload:
        """Encodes an exception into a payload for the network.

        :returns: a payload of type "nil" when exc is None, and a payload of type
            "unknown" that carries the message when no provider handles exc.
        """
        if exc is None:
            return ExceptionPayload(type=_TYPE_NONE, data="")
        for provider in self._providers:
            pld = provider.encode(exc)
            if pld is not None:
                return pld
        return ExceptionPayload(type=_TYPE_UNKNOWN, data=str(exc))

    def decode(self, pld: ExceptionPayload | None) -> Exception | None:
        """Decodes a payload from the network into an exception.

        :returns: None when pld is None or has type "nil", and a generic Exception
            that carries the payload's data when no provider handles its type.
        """
        if pld is None or pld.type == _TYPE_NONE:
            return None
        for provider in self._providers:
            exc = provider.decode(pld)
            if exc is not None:
                return exc
        return Exception(pld.data)


class ContiguityError(Exception):
    """Raised when time-series data is not contiguous."""

    TYPE = "sy.contiguity"
