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

user_ontology_type = ID(type="user")


class NewUser(Payload):
    username: str
    password: str
    first_name: str | None
    last_name: str | None


class UserPayload(Payload):
    key: UUID
    username: str
    first_name: str | None
    last_name: str | None

    def ontology_id(self) -> ID:
        return ID(key=self.key, type="user")
