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
        layout: LayoutClient,
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

        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Force IEPE",
            **kwargs,
        )

        self._configure_dropdown("Force Units", force_units)
        self._configure_input("Sensitivity", sensitivity)

        # Custom handling for sensitivity units (button text contains special chars)
        if sensitivity_units is not None:
            self.layout.page.locator(
                "button.pluto-dialog__trigger:has-text('V/')"
            ).click()
            self.layout.page.locator(".pluto-list__item").get_by_text(
                sensitivity_units, exact=True
            ).click()

        self._configure_dropdown("Current Excitation Source", current_excitation_source)
        self._configure_input("Current Excitation Value", current_excitation_value)
