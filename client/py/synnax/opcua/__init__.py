#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.opcua.types import (
    MAKE,
    MODEL,
    Device,
    ReadTask,
    SecurityMode,
    SecurityPolicy,
    WriteTask,
)
from synnax.opcua.types_gen import (
    BaseChannel,
    ReadChannel,
    ReadConfig,
    ScanConfig,
    WriteChannel,
    WriteConfig,
)
from x.deprecation import deprecated_getattr

_DEPRECATED: dict[str, str | tuple[str, str]] = {
    "Channel": "ReadChannel",
    "WriteTaskConfig": "WriteConfig",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "BaseChannel",
    "Device",
    "MAKE",
    "MODEL",
    "ReadChannel",
    "ReadConfig",
    "ReadTask",
    "ScanConfig",
    "SecurityMode",
    "SecurityPolicy",
    "WriteChannel",
    "WriteConfig",
    "WriteTask",
]
