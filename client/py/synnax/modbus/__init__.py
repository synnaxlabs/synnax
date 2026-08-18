#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.modbus.types import (
    MAKE,
    MODEL,
    Device,
    ReadTask,
    WriteTask,
)
from synnax.modbus.types_gen import (
    BaseReadChannel,
    BaseWriteChannel,
    CoilReadChannel,
    CoilWriteChannel,
    DiscreteInputReadChannel,
    HoldingRegisterReadChannel,
    HoldingRegisterWriteChannel,
    InputRegisterReadChannel,
    ReadChannel,
    ReadConfig,
    RegisterValue,
    ScanConfig,
    WriteChannel,
    WriteConfig,
)
from x.deprecation import deprecated_getattr

_DEPRECATED: dict[str, str | tuple[str, str]] = {
    "InputChan": "ReadChannel",
    "OutputChan": "WriteChannel",
    "CoilInputChan": "CoilReadChannel",
    "CoilOutputChan": "CoilWriteChannel",
    "DiscreteInputChan": "DiscreteInputReadChannel",
    "HoldingRegisterInputChan": "HoldingRegisterReadChannel",
    "HoldingRegisterOutputChan": "HoldingRegisterWriteChannel",
    "InputRegisterChan": "InputRegisterReadChannel",
    "ReadTaskConfig": "ReadConfig",
    "WriteTaskConfig": "WriteConfig",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "BaseReadChannel",
    "BaseWriteChannel",
    "CoilReadChannel",
    "CoilWriteChannel",
    "Device",
    "DiscreteInputReadChannel",
    "HoldingRegisterReadChannel",
    "HoldingRegisterWriteChannel",
    "InputRegisterReadChannel",
    "MAKE",
    "MODEL",
    "ReadChannel",
    "ReadConfig",
    "ReadTask",
    "RegisterValue",
    "ScanConfig",
    "WriteChannel",
    "WriteConfig",
    "WriteTask",
]
