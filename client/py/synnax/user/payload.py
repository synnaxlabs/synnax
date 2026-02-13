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
    first_name: str = ""
    last_name: str = ""
    key: UUID | None = None


class User(BaseModel):
    key: UUID
    username: str
    first_name: str
    last_name: str

    @property
    def ontology_id(self) -> ontology.ID:
        return ontology_id(self.key)


ONTOLOGY_TYPE = ontology.ID(type="user")


def ontology_id(key: UUID) -> ontology.ID:
    """Create an ontology ID for a user.

    Args:
        key: The user key.

    Returns:
        An ontology ID dictionary with type "user" and the given key.
    """
    return ontology.ID(type=ONTOLOGY_TYPE.type, key=str(key))


ACTION_CHANGE_USERNAME = "change_username"
