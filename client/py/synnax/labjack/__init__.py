#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.labjack.types import (
    MAKE,
    SUPPORTED_MODELS,
    T4,
    T7,
    T8,
    Device,
    ReadTask,
    WriteTask,
)
from synnax.labjack.types_gen import (
    AnalogReadChannel,
    AnalogWriteChannel,
    BaseReadChannel,
    BaseWriteChannel,
    DigitalReadChannel,
    DigitalWriteChannel,
    LinearScale,
    MapScale,
    NoneScale,
    ReadChannel,
    ReadConfig,
    Scale,
    ScanConfig,
    TemperatureUnits,
    ThermocoupleReadChannel,
    ThermocoupleType,
    WriteChannel,
    WriteConfig,
)
from x.deprecation import deprecated_getattr

_DEPRECATED: dict[str, str | tuple[str, str]] = {
    "AIChan": "AnalogReadChannel",
    "DIChan": "DigitalReadChannel",
    "ThermocoupleChan": "ThermocoupleReadChannel",
    "OutputChan": "WriteChannel",
    "ReadTaskConfig": "ReadConfig",
    "WriteTaskConfig": "WriteConfig",
}

__getattr__ = deprecated_getattr(__name__, _DEPRECATED, globals())

__all__ = [
    "AnalogReadChannel",
    "AnalogWriteChannel",
    "BaseReadChannel",
    "BaseWriteChannel",
    "Device",
    "DigitalReadChannel",
    "DigitalWriteChannel",
    "LinearScale",
    "MAKE",
    "MapScale",
    "NoneScale",
    "ReadChannel",
    "ReadConfig",
    "ReadTask",
    "SUPPORTED_MODELS",
    "Scale",
    "ScanConfig",
    "T4",
    "T7",
    "T8",
    "TemperatureUnits",
    "ThermocoupleReadChannel",
    "ThermocoupleType",
    "WriteChannel",
    "WriteConfig",
    "WriteTask",
]
