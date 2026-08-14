#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.ethercat.types import (
    MAKE,
    MODEL,
    Device,
    PDOEntry,
    ReadTask,
    WriteTask,
)
from synnax.ethercat.types_gen import (
    AutomaticReadChannel,
    AutomaticWriteChannel,
    BaseReadChannel,
    BaseWriteChannel,
    ManualReadChannel,
    ManualWriteChannel,
    PDOAddress,
    ReadChannel,
    ReadConfig,
    ScanConfig,
    WriteChannel,
    WriteConfig,
)
from x.deprecation import deprecated_getattr

_DEPRECATED: dict[str, str | tuple[str, str]] = {
    "AutomaticInputChan": "AutomaticReadChannel",
    "AutomaticOutputChan": "AutomaticWriteChannel",
    "ManualInputChan": "ManualReadChannel",
    "ManualOutputChan": "ManualWriteChannel",
    "ReadTaskConfig": "ReadConfig",
    "WriteTaskConfig": "WriteConfig",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "AutomaticReadChannel",
    "AutomaticWriteChannel",
    "BaseReadChannel",
    "BaseWriteChannel",
    "Device",
    "MAKE",
    "MODEL",
    "ManualReadChannel",
    "ManualWriteChannel",
    "PDOAddress",
    "PDOEntry",
    "ReadChannel",
    "ReadConfig",
    "ReadTask",
    "ScanConfig",
    "WriteChannel",
    "WriteConfig",
    "WriteTask",
]
