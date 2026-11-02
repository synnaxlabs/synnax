#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from typing import Concatenate, ParamSpec, TypeVar

from console.layout import LayoutClient
from console.page import PageType
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
from console.task.ni import NITask

_Channel = TypeVar(
    "_Channel",
    bound=(
        Accelerometer
        | Bridge
        | Current
        | ForceBridgeTable
        | ForceBridgeTwoPointLinear
        | ForceIEPE
        | Microphone
        | PressureBridgeTable
        | PressureBridgeTwoPointLinear
        | Resistance
        | RTD
        | StrainGauge
        | TemperatureBuiltInSensor
        | Thermocouple
        | TorqueBridgeTable
        | TorqueBridgeTwoPointLinear
        | VelocityIEPE
        | Voltage
    ),
)
_P = ParamSpec("_P")


class AnalogRead(NITask):
    """NI Analog Read/Input Task automation interface."""

    page_type: PageType = "NI analog read task"
    pluto_label: str = ".console-task-configure--ni_analog_read"

    def add_channel(
        self,
        name: str,
        chan_type: Callable[Concatenate[LayoutClient, str, str, _P], _Channel],
        device: str,
        dev_name: str | None = None,
        *args: _P.args,
        **kwargs: _P.kwargs,
    ) -> _Channel:
        """
        Add an analog read channel to the task.

        Args:
            name: Channel name
            chan_type: Channel class, which must be valid for analog read tasks
            device: Device identifier
            dev_name: Optional device name
            *args, **kwargs: Channel-specific configuration

        Returns:
            The created channel instance
        """
        return self._add_channel(name, chan_type, device, dev_name, *args, **kwargs)
