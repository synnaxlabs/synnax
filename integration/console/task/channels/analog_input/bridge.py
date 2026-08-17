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


class Bridge(Analog):
    """
    Bridge channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        units (str): "mV/V" or "V/V"
        configuration (str): "FullBridge", "HalfBridge", or "QuarterBridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", or "None"
        excitation_value (float): Voltage excitation value

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
        units: Literal["mV/V", "V/V"] | None = None,
        configuration: (
            Literal["Full bridge", "Half bridge", "Quarter bridge"] | None
        ) = None,
        resistance: float | None = None,
        excitation_source: Literal["Internal", "External", "None"] | None = None,
        excitation_value: float | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Bridge",
            **kwargs,
        )

        self._configure_dropdown("Electrical units", units)
        self._configure_dropdown("Bridge configuration", configuration)
        self._configure_input("Nominal bridge resistance", resistance)
        self._configure_dropdown("Voltage excitation source", excitation_source)
        self._configure_input("Voltage excitation value", excitation_value)
