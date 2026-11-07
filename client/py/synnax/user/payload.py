#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from pydantic import BaseModel

from synnax import resource
from synnax.user.types_gen import Key


class New(BaseModel):
    username: str
    password: str
    name: str = ""
    key: UUID | None = None


class User(BaseModel):
    key: UUID
    name: str

    @property
    def resource_id(self) -> resource.ID:
        return _resource_id(self.key)


_RESOURCE_TYPE = resource.ID(type="user")


def _resource_id(key: Key) -> resource.ID:
    return resource.ID(type=_RESOURCE_TYPE.type, key=str(key))
