import dataclasses
import enum
import json
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
        return json.dumps(
            dataclasses.asdict(payload),
            ensure_ascii=False,
            default=json_default,
        ).encode()

    @staticmethod
    def decode(data: bytes, payload: any):
        merge_payload_dict(json.loads(data), payload)


def json_default(self, obj: Any) -> Any:
    if isinstance(obj, EncodeableDecodeable):
        return obj.encode()
    return json.JSONEncoder.default(self, obj)


def msgpack_default(self, obj: Any) -> Any:
    if isinstance(obj, EncodeableDecodeable):
        return obj.encode()
    return obj


def merge_payload_dict(data: dict, payload: any):
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
            if isinstance(value, EncodeableDecodeable):
                value = value.decode()
            setattr(payload, key, value)


ENCODER_DECODERS: list[EncoderDecoder] = [JSONEncoderDecoder, MsgpackEncoderDecoder]

T = TypeVar("T")


@runtime_checkable
class EncodeableDecodeable(Protocol[T]):
    def encode(self) -> Any:
        ...

    @staticmethod
    def decode(value: Any) -> T:
        ...
