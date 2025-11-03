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


class ForceIEPE(Analog):
    """
    Force IEPE channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        force_units (str): "Newtons", "Pounds", "Kilograms", etc.
        sensitivity (float): Sensor sensitivity value
        sensitivity_units (str): "mV/N", "mV/lb", etc.
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
        force_units: Literal["Newtons", "Pounds"] | None = None,
        sensitivity: float | None = None,
        sensitivity_units: Literal["mV/N", "mV/lb"] | None = None,
        current_excitation_source: (
            Literal["Internal", "External", "None"] | None
        ) = None,
        current_excitation_value: float | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Force IEPE",
            **kwargs,
        )

        # Force IEPE-specific configurations:
        if force_units is not None:
            console.click_btn("Force Units")
            console.select_from_dropdown(force_units)

        if sensitivity is not None:
            console.fill_input_field("Sensitivity", str(sensitivity))

        if sensitivity_units is not None:

            console.page.locator("button.pluto-dialog__trigger:has-text('V/')").click()
            console.page.locator(f".pluto-list__item").get_by_text(
                sensitivity_units, exact=True
            ).click()

            # console.click_btn("Sensitivity")
            # console.select_from_dropdown(sensitivity_units)

        if current_excitation_source is not None:
            console.click_btn("Current Excitation Source")
            console.select_from_dropdown(current_excitation_source)

        if current_excitation_value is not None:
            console.fill_input_field(
                "Current Excitation Value", str(current_excitation_value)
            )
