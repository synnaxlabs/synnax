#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any

from playwright.sync_api import Page

from console.task.channels.accelerometer import Accelerometer
from console.task.channels.analog import Analog
from console.task.channels.bridge import Bridge
from console.task.channels.current import Current
from console.task.channels.force_bridge_table import ForceBridgeTable
from console.task.channels.force_bridge_two_point_linear import (
    ForceBridgeTwoPointLinear,
)
from console.task.channels.force_iepe import ForceIEPE
from console.task.channels.microphone import Microphone
from console.task.channels.pressure_bridge_table import PressureBridgeTable
from console.task.channels.pressure_bridge_two_point_linear import (
    PressureBridgeTwoPointLinear,
)
from console.task.channels.resistance import Resistance
from console.task.channels.rtd import RTD
from console.task.channels.strain_gauge import StrainGauge
from console.task.channels.temperature_built_in_sensor import TemperatureBuiltInSensor
from console.task.channels.thermocouple import Thermocouple
from console.task.channels.torque_bridge_table import TorqueBridgeTable
from console.task.channels.torque_bridge_two_point_linear import (
    TorqueBridgeTwoPointLinear,
)
from console.task.channels.velocity_iepe import VelocityIEPE
from console.task.channels.voltage import Voltage

from .ni import NITask

if TYPE_CHECKING:
    from console.console import Console

# Valid channel types for NI Analog Read tasks
ANALOG_READ_CHANNEL_TYPES: dict[str, type[Analog]] = {
    "Accelerometer": Accelerometer,
    "Bridge": Bridge,
    "Current": Current,
    "Force Bridge Table": ForceBridgeTable,
    "Force Bridge Two-Point Linear": ForceBridgeTwoPointLinear,
    "Force IEPE": ForceIEPE,
    "Microphone": Microphone,
    "Pressure Bridge Table": PressureBridgeTable,
    "Pressure Bridge Two-Point Linear": PressureBridgeTwoPointLinear,
    "Resistance": Resistance,
    "RTD": RTD,
    "Strain Gauge": StrainGauge,
    "Temperature Built-In Sensor": TemperatureBuiltInSensor,
    "Thermocouple": Thermocouple,
    "Torque Bridge Table": TorqueBridgeTable,
    "Torque Bridge Two-Point Linear": TorqueBridgeTwoPointLinear,
    "Velocity IEPE": VelocityIEPE,
    "Voltage": Voltage,
}


class AnalogRead(NITask):
    """NI Analog Read/Input Task automation interface."""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "NI Analog Read Task"

    def new(self) -> str:
        """Create a new NI AI task page."""
        return super().new()

    def add_channel(
        self,
        name: str,
        chan_type: str,
        device: str,
        dev_name: str | None = None,
        **kwargs: Any,
    ) -> Analog:
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

    def set_parameters(
        self,
        task_name: str | None = None,
        data_saving: bool | None = None,
        auto_start: bool | None = None,
        **kwargs: Any,
    ) -> None:
        """
        Set the parameters for the NI AI task.

        Args:
            task_name: The name of the task.
            sample_rate: The sample rate for the AI task.
            stream_rate: The stream rate for the AI task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
            **kwargs: Additional parameters.
        """
        sample_rate = kwargs.pop("sample_rate", None)
        stream_rate = kwargs.pop("stream_rate", None)

        super().set_parameters(
            task_name=task_name,
            data_saving=data_saving,
            auto_start=auto_start,
            **kwargs,
        )

        if sample_rate is not None:
            self.console.fill_input_field("Sample Rate", str(sample_rate))

        if stream_rate is not None:
            self.console.fill_input_field("Stream Rate", str(stream_rate))
