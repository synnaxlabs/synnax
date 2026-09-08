#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal

from console.layout import LayoutClient
from console.task.channels.analog import Analog


class TorqueBridgeTwoPointLinear(Analog):
    """
    Torque Bridge Two-Point Linear channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        torque_units (str): "Newton meters", "Inch ounces", "Foot pounds"
        bridge_configuration (str): "Full bridge", "Half bridge", "Quarter bridge"
        nominal_bridge_resistance (float): Bridge resistance value
        voltage_excitation_source (str): "Internal", "External", "None"
        voltage_excitation_value (float): Excitation voltage value
        physical_units (str): "Newton meters", "Inch ounces", "Foot pounds"
        electrical_units (str): "mV/V", "V/V"
        physical_value_one (float): First physical calibration point
        physical_value_two (float): Second physical calibration point
        electrical_value_one (float): First electrical calibration point
        electrical_value_two (float): Second electrical calibration point

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
        custom_scale (str): "None", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        torque_units: (
            Literal["Newton meters", "Inch ounces", "Foot pounds"] | None
        ) = None,
        bridge_configuration: (
            Literal["Full bridge", "Half bridge", "Quarter bridge"] | None
        ) = None,
        nominal_bridge_resistance: float | None = None,
        voltage_excitation_source: (
            Literal["Internal", "External", "None"] | None
        ) = None,
        voltage_excitation_value: float | None = None,
        physical_units: (
            Literal["Newton meters", "Inch ounces", "Foot pounds"] | None
        ) = None,
        electrical_units: Literal["mV/V", "V/V"] | None = None,
        physical_value_one: float | None = None,
        physical_value_two: float | None = None,
        electrical_value_one: float | None = None,
        electrical_value_two: float | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Torque bridge two-point linear",
            **kwargs,
        )

        self._configure_dropdown("Torque units", torque_units)
        self._configure_dropdown("Bridge configuration", bridge_configuration)
        self._configure_input("Nominal bridge resistance", nominal_bridge_resistance)
        self._configure_dropdown("Voltage excitation source", voltage_excitation_source)
        self._configure_input("Voltage excitation value", voltage_excitation_value)
        self._configure_dropdown("Physical units", physical_units)
        self._configure_dropdown("Electrical units", electrical_units)
        self._configure_input("Physical value one", physical_value_one)
        self._configure_input("Physical value two", physical_value_two)
        self._configure_input("Electrical value one", electrical_value_one)
        self._configure_input("Electrical value two", electrical_value_two)
