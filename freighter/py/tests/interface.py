from __future__ import annotations

from dataclasses import dataclass
from typing import Type, runtime_checkable

from freighter import errors


@dataclass
class Message:
    id: int | None
    message: str | None

    @classmethod
    def new(cls: Type[Message]) -> Message:
        return Message(id=None, message=None)


@dataclass
class Error(Exception):
    code: int
    message: str


def encode_test_error(exc: Exception) -> str:
    assert isinstance(exc, Error)
    return f"{exc.code},{exc.message}"


def decode_test_error(encoded: str) -> Exception:
    code, message = encoded.split(",")
    return Error(int(code), message)


errors.register("integration.error", encode_test_error, decode_test_error)
