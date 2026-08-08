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
    BaseInputChannel,
    BaseOutputChannel,
    InputChannel,
    InputChannelAutomatic,
    InputChannelManual,
    OutputChannel,
    OutputChannelAutomatic,
    OutputChannelManual,
    PDOAddress,
    ReadConfig,
    ScanConfig,
    WriteConfig,
)

__all__ = [
    "BaseInputChannel",
    "BaseOutputChannel",
    "Device",
    "InputChannel",
    "InputChannelAutomatic",
    "InputChannelManual",
    "MAKE",
    "MODEL",
    "OutputChannel",
    "OutputChannelAutomatic",
    "OutputChannelManual",
    "PDOAddress",
    "PDOEntry",
    "ReadConfig",
    "ReadTask",
    "ScanConfig",
    "WriteConfig",
    "WriteTask",
]
