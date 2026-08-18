#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.task.client import (
    Client,
    JSONConfigMixin,
    Protocol,
    StarterStopperMixin,
    Task,
    assign_keys,
)
from synnax.task.types_gen import (
    ONTOLOGY_TYPE,
    BasePersistConfig,
    BaseReadConfig,
    BaseScanConfig,
    BaseStartConfig,
    BaseWriteConfig,
    Command,
    Key,
    KeyedConfig,
    Payload,
    Status,
    StatusDetails,
    ontology_id,
)
from x.deprecation import deprecated_getattr

_DEPRECATED = {
    "TaskPayload": "Payload",
    "TaskStatus": "Status",
    "TaskStatusDetails": "StatusDetails",
    "BaseTaskConfig": "BaseStartConfig",
    "BaseReadTaskConfig": "BaseReadConfig",
    "BaseWriteTaskConfig": "BaseWriteConfig",
    "TaskProtocol": "Protocol",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "Client",
    "Command",
    "Key",
    "Task",
    "Payload",
    "Status",
    "StatusDetails",
    "ONTOLOGY_TYPE",
    "ontology_id",
    "assign_keys",
    "BasePersistConfig",
    "BaseStartConfig",
    "BaseReadConfig",
    "BaseScanConfig",
    "BaseWriteConfig",
    "KeyedConfig",
    "JSONConfigMixin",
    "StarterStopperMixin",
    "Protocol",
]
