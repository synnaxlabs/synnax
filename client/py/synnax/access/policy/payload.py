#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from x.deprecation import deprecated_getattr

from synnax.access.policy.types_gen import Key, Policy, ontology_id

__all__ = [
    "Key",
    "Policy",
    "ontology_id",
    "ACTION_CREATE",
    "ACTION_DELETE",
    "ACTION_RETRIEVE",
    "ACTION_UPDATE",
]

ACTION_CREATE = "create"
ACTION_DELETE = "delete"
ACTION_RETRIEVE = "retrieve"
ACTION_UPDATE = "update"


_DEPRECATED = {
    "CREATE_ACTION": "ACTION_CREATE",
    "DELETE_ACTION": "ACTION_DELETE",
    "RETRIEVE_ACTION": "ACTION_RETRIEVE",
    "UPDATE_ACTION": "ACTION_UPDATE",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())
