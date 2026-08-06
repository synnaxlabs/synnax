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
    BaseInputChannel,
    BaseOutputChannel,
    InputChannel,
    InputChannelCoilInput,
    InputChannelDiscreteInput,
    InputChannelHoldingRegisterInput,
    InputChannelRegisterInput,
    OutputChannel,
    OutputChannelCoilOutput,
    OutputChannelHoldingRegisterOutput,
    ReadConfig,
    RegisterValue,
    ScanConfig,
    WriteConfig,
)

__all__ = [
    "BaseInputChannel",
    "BaseOutputChannel",
    "Device",
    "InputChannel",
    "InputChannelCoilInput",
    "InputChannelDiscreteInput",
    "InputChannelHoldingRegisterInput",
    "InputChannelRegisterInput",
    "MAKE",
    "MODEL",
    "OutputChannel",
    "OutputChannelCoilOutput",
    "OutputChannelHoldingRegisterOutput",
    "ReadConfig",
    "ReadTask",
    "RegisterValue",
    "ScanConfig",
    "WriteConfig",
    "WriteTask",
]
