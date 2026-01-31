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


class DutyCycle(Counter):
    """
    Duty Cycle channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum frequency
        max_val (float): Maximum frequency
        edge (str): "Rising" or "Falling"
        terminal (str): Input terminal (e.g., "PFI0", "PFI1", etc.)
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        edge: Optional[Literal["Rising", "Falling"]] = None,
        terminal: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize duty cycle channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Duty Cycle",
            **kwargs,
        )
        layout = self.layout

        # Active Edge
        if edge is not None:
            layout.click_btn("Active Edge")
            layout.select_from_dropdown(edge)
            self.form_values["Active Edge"] = edge
        else:
            self.form_values["Active Edge"] = layout.get_dropdown_value("Active Edge")

        # Terminal
        if terminal is not None:
            layout.click_btn("Input Terminal")
            layout.select_from_dropdown(terminal)
            self.form_values["Input Terminal"] = terminal
        else:
            self.form_values["Input Terminal"] = layout.get_dropdown_value(
                "Input Terminal"
            )
