#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from pydantic import (
    BaseModel,
    Field,
    SerializationInfo,
    SerializerFunctionWrapHandler,
    model_serializer,
)
from pydantic_core import to_jsonable_python

from x.patch.types_gen import Guard


class Update[V](BaseModel):
    """Changes a record by a partial record.

    A None in value clears a field, so value keeps every None when a caller dumps the
    update with exclude_none, as the client codec does.

    Attributes:
        value: Is the partial record that the update applies.
        guards: Are the variant guards that must hold in the stored record before the
            update applies.
    """

    value: V
    guards: list[Guard] = Field(default_factory=list)

    @model_serializer(mode="wrap")
    def _keep_clears(
        self, handler: SerializerFunctionWrapHandler, info: SerializationInfo
    ) -> dict[str, Any]:
        data: dict[str, Any] = handler(self)
        data["value"] = to_jsonable_python(self.value, by_alias=bool(info.by_alias))
        return data
