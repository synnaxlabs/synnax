#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


class StrainGauge(Analog):
    """
    Strain Gauge channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        strain_configuration (str): "Full Bridge I", "Full Bridge II", "Full Bridge III", "Half Bridge I", "Half Bridge II", "Quarter Bridge I"
        excitation_source (str): "Internal", "External", "None"
        excitation_value (float): Voltage excitation value
        gage_factor (float): Gage factor value
        initial_bridge_voltage (float): Initial bridge voltage
        nominal_gage_resistance (float): Nominal gage resistance
        poisson_ratio (float): Poisson's ratio
        lead_wire_resistance (float): Lead wire resistance

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
        custom_scale (str): "None", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        strain_configuration: (
            Literal[
                "Full Bridge I",
                "Full Bridge II",
                "Full Bridge III",
                "Half Bridge I",
                "Half Bridge II",
                "Quarter Bridge I",
            ]
            | None
        ) = None,
        excitation_source: Literal["Internal", "External", "None"] | None = None,
        excitation_value: float | None = None,
        gage_factor: float | None = None,
        initial_bridge_voltage: float | None = None,
        nominal_gage_resistance: float | None = None,
        poisson_ratio: float | None = None,
        lead_wire_resistance: float | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Strain Gauge",
            **kwargs,
        )

        # Strain Gauge-specific configurations:
        if strain_configuration is not None:
            console.click_btn("Strain Configuration")
            console.select_from_dropdown(strain_configuration)

        if excitation_source is not None:
            console.click_btn("Voltage Excitation Source")
            console.select_from_dropdown(excitation_source)

        if excitation_value is not None:
            console.fill_input_field("Voltage Excitation Value", str(excitation_value))

        if gage_factor is not None:
            console.fill_input_field("Gage Factor", str(gage_factor))

        if initial_bridge_voltage is not None:
            console.fill_input_field(
                "Initial Bridge Voltage", str(initial_bridge_voltage)
            )

        if nominal_gage_resistance is not None:
            console.fill_input_field(
                "Nominal Gage Resistance", str(nominal_gage_resistance)
            )

        if poisson_ratio is not None:
            console.fill_input_field("Poisson's Ratio", str(poisson_ratio))

        if lead_wire_resistance is not None:
            console.fill_input_field("Lead Wire Resistance", str(lead_wire_resistance))
