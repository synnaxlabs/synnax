from __future__ import annotations

from typing import Type

from pydantic import BaseModel

from freighter import exceptions


class Message(BaseModel):
    id: int | None
    message: str | None

    @classmethod
    def new(cls: Type[Message]) -> Message:
        return Message(id=None, message=None)


class Error(Exception):
    code: int
    message: str

    def __init__(self, code: int, message: str):
        self.code = code
        self.message = message
        super().__init__(message)


def encode_test_error(exc: Exception) -> str:
    assert isinstance(exc, Error)
    return f"{exc.code},{exc.message}"


def decode_test_error(encoded: str) -> Exception:
    code, message = encoded.split(",")
    return Error(int(code), message)


errors.register_exception("integration.error", encode_test_error, decode_test_error)
