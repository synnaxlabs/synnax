import dataclasses
import enum
import json
from typing import Protocol
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
    def encode(payload: any) -> bytes:
        return msgpack.packb(dataclasses.asdict(payload))

    @staticmethod
    def decode(data: bytes, payload: any):
        merge_payload_dict(msgpack.unpackb(data), payload)


class JSONEncoderDecoder:
    @staticmethod
    def content_type():
        return "application/json"

    @staticmethod
    def encode(payload: any) -> bytes:
        return json.dumps(dataclasses.asdict(payload)).encode()

    @staticmethod
    def decode(data: bytes, payload: any):
        merge_payload_dict(json.loads(data), payload)


def merge_payload_dict(data: dict, payload: any):
    is_dict = isinstance(payload, dict)
    for key, value in data.items():
        if isinstance(value, dict):
            sub_payload = payload.get(key) if is_dict else getattr(payload, key)
            merge_payload_dict(value, sub_payload)
        elif is_dict:
            payload[key] = value
        else:
            setattr(payload, key, value)


ENCODERS: list[EncoderDecoder] = [
    JSONEncoderDecoder,
    MsgpackEncoderDecoder
]
