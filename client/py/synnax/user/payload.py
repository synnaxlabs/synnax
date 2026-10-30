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

from synnax import ontology


class New(BaseModel):
    username: str
    password: str
    name: str = ""
    key: UUID | None = None


class User(BaseModel):
    key: UUID
    name: str

    @property
    def ontology_id(self) -> ontology.ID:
        return _ontology_id(self.key)


_ONTOLOGY_TYPE = ontology.ID(type="user")


def _ontology_id(key: UUID) -> ontology.ID:
    return ontology.ID(type=_ONTOLOGY_TYPE.type, key=str(key))
