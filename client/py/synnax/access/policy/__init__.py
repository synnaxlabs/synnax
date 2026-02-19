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
    ACTION_CREATE,
    ACTION_DELETE,
    ACTION_RETRIEVE,
    ACTION_UPDATE,
    Policy,
    ontology_id,
)

# Backwards compatibility
PolicyClient = Client
CREATE_ACTION = ACTION_CREATE
DELETE_ACTION = ACTION_DELETE
RETRIEVE_ACTION = ACTION_RETRIEVE
UPDATE_ACTION = ACTION_UPDATE

__all__ = [
    "Client",
    "Policy",
    "ACTION_CREATE",
    "ACTION_DELETE",
    "ACTION_RETRIEVE",
    "ACTION_UPDATE",
    "ontology_id",
    "PolicyClient",
    "CREATE_ACTION",
    "DELETE_ACTION",
    "RETRIEVE_ACTION",
    "UPDATE_ACTION",
]
