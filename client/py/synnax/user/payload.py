#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from freighter import Payload

from synnax.ontology.payload import ID


class NewUser(Payload):
    username: str
    password: str
    first_name: str = ""
    last_name: str = ""
    key: UUID | None = None


class User(Payload):
    key: UUID
    username: str
    first_name: str
    last_name: str
    root_user: bool

    @property
    def ontology_id(self) -> ID:
        return ID(key=self.key, type="user")


USER_ONTOLOGY_TYPE = ID(type="user")


def ontology_id(key: UUID) -> ID:
    """Create an ontology ID for a user.

    Args:
        key: The user key.

    Returns:
        An ontology ID dictionary with type "user" and the given key.
    """
    return ID(type=USER_ONTOLOGY_TYPE.type, key=str(key))


change_username_action = "change_username"
