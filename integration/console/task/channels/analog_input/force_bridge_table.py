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


class ForceBridgeTable(Analog):
    """
    Force Bridge Table channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        force_units (str): "Newtons", "Pounds", "Kilograms", etc.
        bridge_configuration (str): "Full Bridge", "Half Bridge", "Quarter Bridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", "None"
        excitation_value (float): Voltage excitation value
        physical_units (str): "Newtons", "Pounds", "Kilograms", etc.
        electrical_units (str): "mV/V", "V/V"
        custom_scale (str): "None", "Linear", "Map", "Table"

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        force_units: Literal["Newtons", "Pounds", "Kilograms"] | None = None,
        bridge_configuration: (
            Literal["Full Bridge", "Half Bridge", "Quarter Bridge"] | None
        ) = None,
        resistance: float | None = None,
        excitation_source: Literal["Internal", "External", "None"] | None = None,
        excitation_value: float | None = None,
        physical_units: Literal["Newtons", "Pounds", "Kilograms"] | None = None,
        electrical_units: Literal["mV/V", "V/V"] | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Force Bridge Table",
            **kwargs,
        )
        layout = self.layout

        # Force Bridge Table-specific configurations:
        if force_units is not None:
            layout.click_btn("Force Units")
            layout.select_from_dropdown(force_units)

        if bridge_configuration is not None:
            layout.click_btn("Bridge Configuration")
            layout.select_from_dropdown(bridge_configuration)

        if resistance is not None:
            layout.fill_input_field("Nominal Bridge Resistance", str(resistance))

        if excitation_source is not None:
            layout.click_btn("Voltage Excitation Source")
            layout.select_from_dropdown(excitation_source)

        if excitation_value is not None:
            layout.fill_input_field("Voltage Excitation Value", str(excitation_value))

        if physical_units is not None:
            layout.click_btn("Physical Units")
            layout.select_from_dropdown(physical_units)

        if electrical_units is not None:
            layout.click_btn("Electrical Units")
            layout.select_from_dropdown(electrical_units)
