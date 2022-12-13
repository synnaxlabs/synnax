from typing import Protocol, Type, TypeVar

import msgpack

from .transport import P, Payload



class EncoderDecoder(Protocol):
    """Protocol for an entity that encodes and decodes values from binary.
    """
    @staticmethod
    def content_type() -> str:
        """:returns: the HTTP content type of the encoder"""
        ...

    @staticmethod
    def encode(data: Payload) -> bytes:
        """Encodes the given data into a binary representation.
        :param data: The data to encode.
        :returns: The binary representation of the data.
        """
        ...

    @staticmethod
    def decode(data: bytes, pld_t: Type[P]) -> P:
        """Decodes the given binary into a type checked payload.
        :param data: THe binary to decode.
        :param pld_t: The type of the payload to decode into.
        """
        ...


class MsgpackEncoder:
    """A Msgpack implementation of EncoderDecoder.
    """
    @staticmethod
    def content_type():
        return "application/msgpack"

    @staticmethod
    def encode(payload: Payload) -> bytes:
        return msgpack.packb(payload.dict())

    @staticmethod
    def decode(data: bytes, pld_t: Type[P]) -> P:
        return pld_t.parse_obj(msgpack.unpackb(data))


class JSONEncoder:
    """A JSON implementation of EncoderDecoder.
    """
    STRING_ENCODING = "utf-8"

    @staticmethod
    def content_type():
        return "application/json"

    @staticmethod
    def encode(payload: Payload) -> bytes:
        return payload.json().encode()

    @staticmethod
    def decode(data: bytes, pld_t: Type[P]) -> P:
        return pld_t.parse_raw(data.decode(JSONEncoder.STRING_ENCODING))


ENCODER_DECODERS: list[EncoderDecoder] = [
    JSONEncoder(),
    MsgpackEncoder(),
]

T = TypeVar("T")
