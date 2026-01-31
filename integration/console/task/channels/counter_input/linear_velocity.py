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
from console.task.channels.counter import Counter


class LinearVelocity(Counter):
    """
    Linear Velocity channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        units (str): "m/s" or "in/s"
        decoding_type (str): "X1", "X2", "X4", or "Two Pulse"
        dist_per_pulse (float): Distance per pulse
        terminal_a (str): Input Terminal A
        terminal_b (str): Input Terminal B
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        units: Literal["m/s", "in/s"] | None = None,
        decoding_type: Literal["X1", "X2", "X4", "Two Pulse"] | None = None,
        dist_per_pulse: float | None = None,
        terminal_a: str | None = None,
        terminal_b: str | None = None,
        **kwargs: Any,
    ) -> None:
        """Initialize linear velocity channel with configuration."""
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Velocity Linear",
            **kwargs,
        )
        layout = self.layout

        # Units
        if units is not None:
            layout.click_btn("Scaled Units")
            layout.select_from_dropdown(units)
            self.form_values["Scaled Units"] = units
        else:
            self.form_values["Scaled Units"] = layout.get_dropdown_value("Scaled Units")

        # Distance per Pulse
        if dist_per_pulse is not None:
            layout.fill_input_field("Distance / Pulse", str(dist_per_pulse))
            self.form_values["Distance / Pulse"] = str(dist_per_pulse)
        else:
            self.form_values["Distance / Pulse"] = layout.get_input_field(
                "Distance / Pulse"
            )

        # Decoding Type
        if decoding_type is not None:
            layout.click_btn("Decoding Type")
            layout.select_from_dropdown(decoding_type)
            self.form_values["Decoding Type"] = decoding_type
        else:
            self.form_values["Decoding Type"] = layout.get_dropdown_value(
                "Decoding Type"
            )

        # Input Terminal A
        if terminal_a is not None:
            layout.click_btn("Input Terminal A")
            layout.select_from_dropdown(terminal_a)
            self.form_values["Input Terminal A"] = terminal_a
        else:
            self.form_values["Input Terminal A"] = layout.get_dropdown_value(
                "Input Terminal A"
            )

        # Input Terminal B
        if terminal_b is not None:
            layout.click_btn("Input Terminal B")
            layout.select_from_dropdown(terminal_b)
            self.form_values["Input Terminal B"] = terminal_b
        else:
            self.form_values["Input Terminal B"] = layout.get_dropdown_value(
                "Input Terminal B"
            )
