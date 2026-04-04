#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from __future__ import annotations

from uuid import UUID

from pydantic import BaseModel

from synnax import ontology
from synnax.access.types_gen import Action
from synnax.util.deprecation import deprecated_getattr

ACTION_CREATE: Action = "create"
ACTION_DELETE: Action = "delete"
ACTION_RETRIEVE: Action = "retrieve"
ACTION_UPDATE: Action = "update"


class Policy(BaseModel):
    key: UUID | None = None
    name: str
    objects: list[ontology.ID] = []
    actions: list[str] = []
    internal: bool = False


def ontology_id(key: UUID | None = None) -> ontology.ID:
    return ontology.ID(type="policy", key=key if key is None else str(key))


_DEPRECATED = {
    "CREATE_ACTION": "ACTION_CREATE",
    "DELETE_ACTION": "ACTION_DELETE",
    "RETRIEVE_ACTION": "ACTION_RETRIEVE",
    "UPDATE_ACTION": "ACTION_UPDATE",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())
