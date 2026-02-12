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
    """
    ExceptionPayload is a payload that can be sent between a Freighter client and
    server, so that it can be decoded into a proper exception by the implementing
    language.
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
    """
    Registers a custom error encoder and decoder with the Freighter error registry,
    which allows it to be sent over the network.

    :param _encode: A function that takes an exception and returns a string.
    :param _decode: A function that takes a string and returns an exception.
    :return: None
    """
    REGISTRY.register(_ExceptionProvider(_encode, _decode))


def encode_exception(exc: Exception) -> ExceptionPayload:
    """
    Encodes an exception into a payload that can be sent between a Freighter server and
    client.

    :param exc: The exception to encode.
    :return: The encoded error payload.
    """
    return REGISTRY.encode(exc)


def decode_exception(encoded: ExceptionPayload | None) -> Exception | None:
    """
    Decode decodes an error payload into an exception. If a custom decoder can be found
    for the error type, it will be used. Otherwise, a generic Exception containing the
    error data is returned.

    :param encoded: The encoded error payload.
    :return: The decoded exception.
    """
    return None if encoded is None else REGISTRY.decode(encoded)


_FREIGHTER_EXCEPTION_TYPE = "freighter."


class Unreachable(Exception):
    """
    Raised when a target is unreachable.
    """

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "unreachable"

    target: str
    message: str

    def __init__(self, target: str = "", message: str = "Unreachable"):
        self.target = target
        self.message = message
        super().__init__(message)

    def __str__(self) -> str:
        return self.message


class StreamClosed(Exception):
    """
    Raised when a stream is closed.
    """

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "stream_closed"

    def __str__(self) -> str:
        return "StreamClosed"


class EOF(Exception):
    """
    Raised when a stream is closed.
    """

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "eof"

    def __str__(self) -> str:
        return "EOF"


def _freighter_encode(exc: Exception) -> ExceptionPayload | None:
    if isinstance(exc, Unreachable):
        return ExceptionPayload(
            type=Unreachable.TYPE,
            data=exc.message,
        )
    if isinstance(exc, StreamClosed):
        return ExceptionPayload(type=StreamClosed.TYPE, data=str(exc))
    if isinstance(exc, EOF):
        return ExceptionPayload(type=EOF.TYPE, data=str(exc))
    return None


def _freighter_decode(exc: ExceptionPayload) -> Exception | None:
    if exc.type is None or not exc.type.startswith(_FREIGHTER_EXCEPTION_TYPE):
        return None
    if exc.type == Unreachable.TYPE:
        return Unreachable(message=exc.data) if exc.data is not None else Unreachable()
    if exc.type == StreamClosed.TYPE:
        return StreamClosed()
    if exc.type == EOF.TYPE:
        return EOF()
    raise ValueError(f"Unknown error type: {exc.type}")


register_exception(_freighter_encode, _freighter_decode)
