#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from __future__ import annotations

from uuid import UUID

from freighter import Payload

from synnax.ontology.payload import ID

CREATE_ACTION = "create"
DELETE_ACTION = "delete"
RETRIEVE_ACTION = "retrieve"
UPDATE_ACTION = "update"

ALL_ACTION = "*"
CREATE_ACTION = "create"
DELETE_ACTION = "delete"
RETRIEVE_ACTION = "retrieve"
UPDATE_ACTION = "update"


class Policy(Payload):
    key: UUID | None = None
    name: str
    objects: list[ID] = []
    actions: list[str] = []
    internal: bool = False


def ontology_id(key: UUID | None = None) -> ID:
    return ID(type="policy", key=key if key is None else str(key))
