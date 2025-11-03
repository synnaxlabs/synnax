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


class VelocityIEPE(Analog):
    """
    Velocity IEPE channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        velocity_units (str): "m/s", "in/s"
        sensitivity (float): Velocity sensitivity value
        sensitivity_units (str): "mV/mm/s", "mV/in/s"
        current_excitation_source (str): "Internal", "External", "None"
        current_excitation_value (float): Current excitation value

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
        velocity_units: Literal["m/s", "in/s"] | None = None,
        sensitivity: float | None = None,
        sensitivity_units: Literal["mV/mm/s", "mV/in/s"] | None = None,
        current_excitation_source: (
            Literal["Internal", "External", "None"] | None
        ) = None,
        current_excitation_value: float | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Velocity IEPE",
            **kwargs,
        )

        if velocity_units is not None:
            console.click_btn("Velocity Units")
            console.select_from_dropdown(velocity_units)

        if sensitivity is not None:
            console.fill_input_field("Sensitivity", str(sensitivity))

        if sensitivity_units is not None:
            console.page.locator("button.pluto-dialog__trigger:has-text('mV/')").click()
            console.page.locator(f".pluto-list__item").get_by_text(
                sensitivity_units, exact=True
            ).click()

        if current_excitation_source is not None:
            console.click_btn("Current Excitation Source")
            console.select_from_dropdown(current_excitation_source)

        if current_excitation_value is not None:
            console.fill_input_field(
                "Current Excitation Value", str(current_excitation_value)
            )
