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


class PulseWidth(Counter):
    """
    Pulse Width channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        starting_edge (str): "Rising" or "Falling"
        units (str): "Seconds", "Ticks", or "Custom"
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        starting_edge: Optional[Literal["Rising", "Falling"]] = None,
        units: Optional[Literal["Seconds", "Ticks", "Custom"]] = None,
        terminal: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize pulse width channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            type="Pulse Width",
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

        # Scaled Units
        if units is not None:
            console.click_btn("Scaled Units")
            console.select_from_dropdown(units)
            self.form_values["Scaled Units"] = units
        else:
            self.form_values["Scaled Units"] = console.get_dropdown_value(
                "Scaled Units"
            )

        # Terminal
        if terminal is not None:
            console.click_btn("Input Terminal")
            console.select_from_dropdown(terminal)
            self.form_values["Input Terminal"] = terminal
        else:
            self.form_values["Input Terminal"] = console.get_dropdown_value(
                "Input Terminal"
            )
