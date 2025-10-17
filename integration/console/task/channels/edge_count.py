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


class EdgeCount(Counter):
    """
    Edge Count channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        active_edge (str): "Rising" or "Falling"
        count_direction (str): "CountUp", "CountDown", "ExternallyControlled"
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
        initial_count (int): Initial count value
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        active_edge: Optional[Literal["Rising", "Falling"]] = None,
        count_direction: Optional[
            Literal["Count Up", "Count Down", "Externally Controlled"]
        ] = None,
        terminal: Optional[str] = None,
        initial_count: Optional[int] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize edge count channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            type="Edge Count",
            **kwargs,
        )

        # Active Edge
        if active_edge is not None:
            console.click_btn("Active Edge")
            console.select_from_dropdown(active_edge)
            self.form_values["Active Edge"] = active_edge
        else:
            self.form_values["Active Edge"] = console.get_dropdown_value("Active Edge")

        # Count Direction
        if count_direction is not None:
            console.click_btn("Count Direction")
            console.select_from_dropdown(count_direction)
            self.form_values["Count Direction"] = count_direction
        else:
            self.form_values["Count Direction"] = console.get_dropdown_value(
                "Count Direction"
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

        # Initial Count
        if initial_count is not None:
            console.fill_input_field("Initial Count", str(initial_count))
            self.form_values["Initial Count"] = str(initial_count)
        else:
            self.form_values["Initial Count"] = console.get_input_field("Initial Count")
