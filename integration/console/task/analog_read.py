#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from console.task.channels.analog import Analog
from console.task.channels.analog_input import (
    RTD,
    Accelerometer,
    Bridge,
    Current,
    ForceBridgeTable,
    ForceBridgeTwoPointLinear,
    ForceIEPE,
    Microphone,
    PressureBridgeTable,
    PressureBridgeTwoPointLinear,
    Resistance,
    StrainGauge,
    TemperatureBuiltInSensor,
    Thermocouple,
    TorqueBridgeTable,
    TorqueBridgeTwoPointLinear,
    VelocityIEPE,
    Voltage,
)
from console.task.ni import NIChannel, NITask

# Valid channel types for NI Analog Read tasks
ANALOG_READ_CHANNEL_TYPES: dict[str, type[Analog]] = {
    "Accelerometer": Accelerometer,
    "Bridge": Bridge,
    "Current": Current,
    "Force bridge table": ForceBridgeTable,
    "Force bridge two-point linear": ForceBridgeTwoPointLinear,
    "Force IEPE": ForceIEPE,
    "Microphone": Microphone,
    "Pressure bridge table": PressureBridgeTable,
    "Pressure bridge two-point linear": PressureBridgeTwoPointLinear,
    "Resistance": Resistance,
    "RTD": RTD,
    "Strain gauge": StrainGauge,
    "Temperature built-in sensor": TemperatureBuiltInSensor,
    "Thermocouple": Thermocouple,
    "Torque bridge table": TorqueBridgeTable,
    "Torque bridge two-point linear": TorqueBridgeTwoPointLinear,
    "Velocity IEPE": VelocityIEPE,
    "Voltage": Voltage,
}


class AnalogRead(NITask):
    """NI Analog Read/Input Task automation interface."""

    page_type = "NI analog read task"
    pluto_label: str = ".console-task-configure--ni_analog_read"

    def add_channel(
        self,
        name: str,
        chan_type: str,
        device: str,
        dev_name: str | None = None,
        **kwargs: Any,
    ) -> NIChannel:
        """
        Add an analog read channel to the task.

        Args:
            name: Channel name
            chan_type: Channel type (must be valid for analog read tasks)
            device: Device identifier
            dev_name: Optional device name
            **kwargs: Additional channel-specific configuration

        Returns:
            The created channel instance

        Raises:
            ValueError: If channel type is not valid for analog read tasks
        """
        if chan_type not in ANALOG_READ_CHANNEL_TYPES:
            raise ValueError(
                f"Invalid channel type for NI Analog Read: {chan_type}. "
                f"Valid types: {list(ANALOG_READ_CHANNEL_TYPES.keys())}"
            )

        return self._add_channel_helper(
            name=name,
            device=device,
            dev_name=dev_name,
            channel_class=ANALOG_READ_CHANNEL_TYPES[chan_type],
            **kwargs,
        )
