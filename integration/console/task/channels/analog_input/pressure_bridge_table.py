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


class PressureBridgeTable(Analog):
    """
    Pressure Bridge Table channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        pressure_units (str): "Pascals", "PSI", etc.
        bridge_configuration (str): "Full Bridge", "Half Bridge", "Quarter Bridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", "None"
        excitation_value (float): Voltage excitation value
        physical_units (str): "Pascals", "PSI", etc.
        electrical_units (str): "mV/V", "V/V"

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
        pressure_units: Literal["Pascals", "PSI"] | None = None,
        bridge_configuration: (
            Literal["Full Bridge", "Half Bridge", "Quarter Bridge"] | None
        ) = None,
        resistance: float | None = None,
        excitation_source: Literal["Internal", "External", "None"] | None = None,
        excitation_value: float | None = None,
        physical_units: Literal["Pascals", "PSI"] | None = None,
        electrical_units: Literal["mV/V", "V/V"] | None = None,
        **kwargs: Any,
    ) -> None:

        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Pressure Bridge Table",
            **kwargs,
        )

        self._configure_dropdown("Pressure Units", pressure_units)
        self._configure_dropdown("Bridge Configuration", bridge_configuration)
        self._configure_input("Nominal Bridge Resistance", resistance)
        self._configure_dropdown("Voltage Excitation Source", excitation_source)
        self._configure_input("Voltage Excitation Value", excitation_value)
        self._configure_dropdown("Physical Units", physical_units)
        self._configure_dropdown("Electrical Units", electrical_units)
