#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass
from typing import Callable

from freighter.transport import Payload

_TYPE_UNKNOWN = "unknown"
_TYPE_NONE = "nil"


class ExceptionPayload(Payload):
    """Error payload is a payload that can be sent between a freighter client and server,
    so that it can be decoded into a proper exception by the implementing language.
    """

    type: str | None
    data: str | None


EncoderFunc = Callable[[Exception], ExceptionPayload | None]
DecoderFunc = Callable[[ExceptionPayload], Exception | None]


@dataclass
class _ExceptionProvider:
    encode: EncoderFunc
    decode: DecoderFunc


class _Registry:
    providers: list[_ExceptionProvider]

    def __init__(self):
        self.providers = list()

    def register(self, provider: _ExceptionProvider) -> None:
        self.providers.append(provider)

    @staticmethod
    def encode(error: Exception | None) -> ExceptionPayload:
        raise NotImplemented

    def decode(self, encoded: ExceptionPayload) -> Exception | None:
        assert isinstance(encoded, ExceptionPayload)
        if encoded.type == _TYPE_NONE:
            return None
        for provider in self.providers:
            decoded = provider.decode(encoded)
            if decoded is not None:
                return decoded
        return Exception(encoded.data)


REGISTRY = _Registry()


def register_exception(_encode: EncoderFunc, _decode: DecoderFunc) -> None:
    """Registers a custom error encoder and decoder with the freighter error registry,
    which allows
    it to be sent over the network.

    :param _encode: A function that takes an exception and returns a string.
    :param _decode: A function that takes a string and returns an exception.
    :return: None
    """
    REGISTRY.register(_ExceptionProvider(_encode, _decode))


def encode_exception(exc: Exception) -> ExceptionPayload:
    """Encodes an exception into a payload that can be sent between a freighter server
    and client.

    :param exc: The exception to encode.
    :return: The encoded error payload.
    """
    return REGISTRY.encode(exc)


def decode_exception(encoded: ExceptionPayload | None) -> Exception | None:
    """Decode decodes an error payload into an exception. If a custom decoder can be found
    for the error type, it will be used. Otherwise, a generic Exception containing the
    error data is returned.

    :param encoded: The encoded error payload.
    :return: The decoded exception.
    """
    return None if encoded is None else REGISTRY.decode(encoded)


_FREIGHTER_EXCEPTION_TYPE = "freighter."


class Unreachable(Exception):
    """
    Raise when a target is unreachable.
    """

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "unreachable"

    target: str
    message: str

    def __init__(self, target: str = "", message="Unreachable"):
        self.target = target
        self.message = message
        super().__init__(message)

    def __str__(self):
        return self.message


class StreamClosed(Exception):
    """
    Raised when a stream is closed.
    """

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "stream_closed"

    def __str__(self):
        return "StreamClosed"


class EOF(Exception):
    """
    Raised when a stream is closed.
    """

    TYPE = _FREIGHTER_EXCEPTION_TYPE + "eof"

    def __str__(self):
        return "EOF"


_EXCEPTIONS = [
    Unreachable,
    StreamClosed,
    EOF,
]


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
    if not exc.type.startswith(_FREIGHTER_EXCEPTION_TYPE):
        return None
    if exc.type == Unreachable.TYPE:
        return Unreachable(message=exc.data)
    if exc.type == StreamClosed.TYPE:
        return StreamClosed()
    if exc.type == EOF.TYPE:
        return EOF()
    raise ValueError(f"Unknown error type: {exc.type}")


register_exception(_freighter_encode, _freighter_decode)
