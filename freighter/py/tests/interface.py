#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Type

from pydantic import BaseModel

from freighter import register_exception


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


register_exception("integration.error", encode_test_error, decode_test_error)
