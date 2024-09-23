#  Copyright 2024 Synnax Labs, Inc.
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

    def ontology_id(self) -> ID:
        return ID(key=self.key, type="user")


ontology_type = "user"
change_username_action = "change_username"
