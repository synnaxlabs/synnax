#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.serial.types import MAKE, Device, ReadTask, WriteTask
from synnax.serial.types_gen import (
    FLOW_CONTROL_HARDWARE,
    FLOW_CONTROL_NONE,
    FLOW_CONTROL_SOFTWARE,
    PARITY_EVEN,
    PARITY_MARK,
    PARITY_NONE,
    PARITY_ODD,
    PARITY_SPACE,
    STOP_BITS_ONE,
    STOP_BITS_ONE_AND_HALF,
    STOP_BITS_TWO,
    FlowControl,
    Parity,
    Properties,
    ReadConfig,
    ScanConfig,
    StopBits,
    WriteConfig,
)

__all__ = [
    "Device",
    "FLOW_CONTROL_HARDWARE",
    "FLOW_CONTROL_NONE",
    "FLOW_CONTROL_SOFTWARE",
    "FlowControl",
    "MAKE",
    "PARITY_EVEN",
    "PARITY_MARK",
    "PARITY_NONE",
    "PARITY_ODD",
    "PARITY_SPACE",
    "Parity",
    "Properties",
    "ReadConfig",
    "ReadTask",
    "STOP_BITS_ONE",
    "STOP_BITS_ONE_AND_HALF",
    "STOP_BITS_TWO",
    "ScanConfig",
    "StopBits",
    "WriteConfig",
    "WriteTask",
]
