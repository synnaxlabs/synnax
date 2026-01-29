#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal

from console.task.channels.counter import Counter

if TYPE_CHECKING:
    from console.console import Console


class AngularVelocity(Counter):
    """
    Angular Velocity channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        min_val (float): Minimum value
        max_val (float): Maximum value
        units (str): "RPM", "Radians/s", or "Degrees/s"
        decoding_type (str): "X1", "X2", "X4", or "Two Pulse"
        pulses_per_rev (int): Pulses per revolution
        terminal_a (str): Input Terminal A
        terminal_b (str): Input Terminal B
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        units: Literal["RPM", "Radians/s", "Degrees/s"] | None = None,
        decoding_type: Literal["X1", "X2", "X4", "Two Pulse"] | None = None,
        pulses_per_rev: int | None = None,
        terminal_a: str | None = None,
        terminal_b: str | None = None,
        **kwargs: Any,
    ) -> None:
        """Initialize angular velocity channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Velocity Angular",
            **kwargs,
        )
        self.layout = console.layout
        layout = self.layout

        # Units
        if units is not None:
            layout.click_btn("Scaled Units")
            layout.select_from_dropdown(units)
            self.form_values["Scaled Units"] = units
        else:
            self.form_values["Scaled Units"] = layout.get_dropdown_value("Scaled Units")

        # Pulses per Revolution
        if pulses_per_rev is not None:
            layout.fill_input_field("Pulses / Rev", str(pulses_per_rev))
            self.form_values["Pulses / Rev"] = str(pulses_per_rev)
        else:
            self.form_values["Pulses / Rev"] = layout.get_input_field("Pulses / Rev")

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
