import dataclasses
import enum
import json
import typing
from typing import Protocol, Callable, Any, runtime_checkable, Type, TypeVar
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
        merge_payload_dict(msgpack.unpackb(data), payload)


class JSONEncoderDecoder:
    @staticmethod
    def content_type():
        return "application/json"

    @staticmethod
    def encode(payload: Any) -> bytes:
        return json.dumps(
            dataclasses.asdict(payload),
            ensure_ascii=False,
            default=json_default,
        ).encode()

    @staticmethod
    def decode(data: bytes, payload: Any):
        merge_payload_dict(json.loads(data), payload)


def json_default(obj: Any) -> Any:
    return json.JSONEncoder().default(obj)


def msgpack_default(obj: Any) -> Any:
    return obj


def merge_payload_dict(data: dict, payload: Any):
    is_dict = isinstance(payload, dict)
    for key, value in data.items():
        if isinstance(value, dict):
            sub_payload = payload.get(key) if is_dict else getattr(payload, key)
            merge_payload_dict(value, sub_payload)
        elif isinstance(value, list):
            sub_payload = payload.get(key) if is_dict else getattr(payload, key)
            for i in range(len(value)):
                merge_payload_dict(value[i], sub_payload[i])
        elif is_dict:
            payload[key] = value
        else:
            setattr(payload, key, value)


ENCODER_DECODERS: list[EncoderDecoder] = [
    JSONEncoderDecoder(),
    MsgpackEncoderDecoder(),
]

T = TypeVar("T")
