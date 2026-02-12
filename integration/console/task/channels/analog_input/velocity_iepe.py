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
        layout: LayoutClient,
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
            layout=layout,
            name=name,
            device=device,
            chan_type="Velocity IEPE",
            **kwargs,
        )

        self._configure_dropdown("Velocity Units", velocity_units)
        self._configure_input("Sensitivity", sensitivity)

        # Custom handling for sensitivity units (button text contains special chars)
        if sensitivity_units is not None:
            self.layout.page.locator(
                "button.pluto-dialog__trigger:has-text('mV/')"
            ).click()
            self.layout.page.locator(".pluto-list__item").get_by_text(
                sensitivity_units, exact=True
            ).click()

        self._configure_dropdown("Current Excitation Source", current_excitation_source)
        self._configure_input("Current Excitation Value", current_excitation_value)
