#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import cast, override

from pydantic import BaseModel

from synnax.resource.types_gen import Type


class ID(BaseModel):
    key: str | None = ""
    type: Type

    def __init__(
        self,
        key: CrudeID | None = None,
        *,
        type: Type | None = None,
    ):
        if isinstance(key, ID):
            super().__init__(key=key.key, type=key.type)
        elif type is None and key is not None:
            type_str, key = key.split(":")
            # cast satisfies mypy; pydantic validates the value against
            # the Type literal at runtime.
            super().__init__(key=key, type=cast(Type, type_str))
        else:
            super().__init__(key=key, type=type)

    @override
    def __str__(self) -> str:
        return f"{self.type}:{self.key}"


CrudeID = str | ID
