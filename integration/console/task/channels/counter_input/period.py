#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, Optional

from console.layout import LayoutClient
from console.task.channels.counter import Counter


class Period(Counter):
    """
    Period channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        starting_edge (str): "Rising" or "Falling"
        units (str): "Seconds", "Ticks", or "Custom"
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
        meas_method (str): "One Counter (Low Frequency)", "Two Counters (High Frequency)", "Two Counters (Large Range)", "Dynamic Averaging"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        starting_edge: Optional[Literal["Rising", "Falling"]] = None,
        units: Optional[Literal["Seconds", "Ticks", "Custom"]] = None,
        terminal: Optional[str] = None,
        meas_method: Optional[
            Literal[
                "One Counter (Low Frequency)",
                "Two Counters (High Frequency)",
                "Two Counters (Large Range)",
                "Dynamic Averaging",
            ]
        ] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize period channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Period",
            **kwargs,
        )
        layout = self.layout

        # Starting Edge
        if starting_edge is not None:
            layout.click_btn("Starting Edge")
            layout.select_from_dropdown(starting_edge)
            self.form_values["Starting Edge"] = starting_edge
        else:
            self.form_values["Starting Edge"] = layout.get_dropdown_value(
                "Starting Edge"
            )

        # Units
        if units is not None:
            layout.click_btn("Units")
            layout.select_from_dropdown(units)
            self.form_values["Units"] = units
        else:
            self.form_values["Units"] = layout.get_dropdown_value("Units")

        # Terminal
        if terminal is not None:
            layout.click_btn("Input Terminal")
            layout.select_from_dropdown(terminal)
            self.form_values["Input Terminal"] = terminal
        else:
            self.form_values["Input Terminal"] = layout.get_dropdown_value(
                "Input Terminal"
            )

        # Measurement Method
        if meas_method is not None:
            layout.click_btn("Measurement Method")
            layout.select_from_dropdown(meas_method)
            self.form_values["Measurement Method"] = meas_method
        else:
            self.form_values["Measurement Method"] = layout.get_dropdown_value(
                "Measurement Method"
            )
