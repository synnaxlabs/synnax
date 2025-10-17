#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal, Optional

from console.task.channels.counter import Counter

if TYPE_CHECKING:
    from console.console import Console


class Period(Counter):
    """
    Period channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        starting_edge (str): "Rising" or "Falling"
        units (str): "Seconds", "Ticks", or "FromCustomScale"
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
        meas_method (str): "1 Counter (Low Frequency)", "2 Counters (High Frequency)", "2 Counters (Large Range)", "Dynamic Averaging"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        starting_edge: Optional[Literal["Rising", "Falling"]] = None,
        units: Optional[Literal["Seconds", "Ticks", "FromCustomScale"]] = None,
        terminal: Optional[str] = None,
        meas_method: Optional[
            Literal[
                "1 Counter (Low Frequency)",
                "2 Counters (High Frequency)",
                "2 Counters (Large Range)",
                "Dynamic Averaging",
            ]
        ] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize period channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            type="Period",
            **kwargs,
        )

        # Starting Edge
        if starting_edge is not None:
            console.click_btn("Starting Edge")
            console.select_from_dropdown(starting_edge)
            self.form_values["Starting Edge"] = starting_edge
        else:
            self.form_values["Starting Edge"] = console.get_dropdown_value(
                "Starting Edge"
            )

        # Units
        if units is not None:
            console.click_btn("Units")
            console.select_from_dropdown(units)
            self.form_values["Units"] = units
        else:
            self.form_values["Units"] = console.get_dropdown_value("Units")

        # Terminal
        if terminal is not None:
            console.click_btn("Input Terminal")
            console.select_from_dropdown(terminal)
            self.form_values["Input Terminal"] = terminal
        else:
            self.form_values["Input Terminal"] = console.get_dropdown_value(
                "Input Terminal"
            )

        # Measurement Method
        if meas_method is not None:
            console.click_btn("Measurement Method")
            console.select_from_dropdown(meas_method)
            self.form_values["Measurement Method"] = meas_method
        else:
            self.form_values["Measurement Method"] = console.get_dropdown_value(
                "Measurement Method"
            )
