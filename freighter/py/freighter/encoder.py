from typing import Protocol, Type, TypeVar

import msgpack

from .transport import P, Payload


class EncoderDecoder(Protocol):
    @staticmethod
    def content_type() -> str:
        ...

    @staticmethod
    def encode(data: Payload) -> bytes:
        ...

    @staticmethod
    def decode(data: bytes, pld_t: Type[P]) -> P:
        ...


class MsgpackEncoder:
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
