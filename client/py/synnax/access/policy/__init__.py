#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.access.policy.client import Client
from synnax.access.policy.payload import (
    Policy,
    ontology_id,
)
from synnax.access.types_gen import Action
from synnax.util.deprecation import deprecated_getattr

_DEPRECATED = {
    "PolicyClient": "Client",
    "ACTION_CREATE": "Action.CREATE",
    "ACTION_DELETE": "Action.DELETE",
    "ACTION_RETRIEVE": "Action.RETRIEVE",
    "ACTION_UPDATE": "Action.UPDATE",
    "CREATE_ACTION": "Action.CREATE",
    "DELETE_ACTION": "Action.DELETE",
    "RETRIEVE_ACTION": "Action.RETRIEVE",
    "UPDATE_ACTION": "Action.UPDATE",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "Action",
    "Client",
    "Policy",
    "ontology_id",
]
