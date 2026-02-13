#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from pydantic import BaseModel

from freighter import ExceptionPayload, register_exception


class Message(BaseModel):
    id: int | None
    message: str | None

    @classmethod
    def new(cls: type[Message]) -> Message:
        return Message(id=None, message=None)


class Error(Exception):
    code: int
    message: str

    def __init__(self, code: int, message: str):
        self.code = code
        self.message = message
        super().__init__(message)


def encode_test_error(exc: Exception) -> ExceptionPayload:
    if not isinstance(exc, Error):
        raise TypeError
    assert isinstance(exc, Error)
    return ExceptionPayload(type="integration.error", data=f"{exc.code},{exc.message}")


def decode_test_error(encoded: ExceptionPayload) -> Exception | None:
    if encoded.type != "integration.error":
        return None
    data = encoded.data
    assert data is not None
    code, message = data.split(",")
    return Error(int(code), message)


register_exception(encode_test_error, decode_test_error)
