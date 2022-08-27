import dataclasses
import json
from typing import Protocol, Any, TypeVar, runtime_checkable
from .transport import Payload

import msgpack


class EncoderDecoder(Protocol):
    @staticmethod
    def content_type() -> str:
        ...

    @staticmethod
    def encode(data: Payload) -> bytes:
        ...

    @staticmethod
    def decode(data: bytes, payload: Payload):
        ...


class MsgpackEncoderDecoder:
    @staticmethod
    def content_type():
        return "application/msgpack"

    @staticmethod
    def encode(payload: Any) -> bytes:
        return msgpack.packb(dataclasses.asdict(payload))

    @staticmethod
    def decode(data: bytes, payload: Any):
        parse_payload_dict(msgpack.unpackb(data), payload)


class JSONEncoderDecoder:
    @staticmethod
    def content_type():
        return "application/json"

    @staticmethod
    def encode(payload: Any) -> bytes:
        return json.dumps(dataclasses.asdict(payload)).encode()

    @staticmethod
    def decode(data: bytes, payload: Any):
        parse_payload_dict(json.loads(data), payload)


def parse_payload_dict(data: dict, payload: Any):
    if isinstance(payload, Loadable):
        payload.load(data)
        return

    is_dict = isinstance(payload, dict)
    for key, value in data.items():
        if isinstance(value, dict):
            sub_payload = payload.get(key) if is_dict else getattr(payload, key)
            parse_payload_dict(value, sub_payload)
        elif is_dict:
            payload[key] = value
        else:
            setattr(payload, key, value)


ENCODER_DECODERS: list[EncoderDecoder] = [
    JSONEncoderDecoder(),
    MsgpackEncoderDecoder(),
]

T = TypeVar("T")


@runtime_checkable
class Loadable(Protocol):
    def load(self, data: dict):
        ...
