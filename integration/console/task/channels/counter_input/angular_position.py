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


class AngularPosition(Counter):
    """
    Angular Position channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        units (str): "Degrees", "Radians", or "Ticks"
        decoding_type (str): "X1", "X2", "X4", or "Two Pulse"
        pulses_per_rev (int): Pulses per revolution
        initial_angle (float): Initial angle value
        z_index_enable (bool): Enable Z index
        z_index_val (float): Z index value
        z_index_phase (str): "A High B High", "A High B Low", "A Low B High", or "A Low B Low"
        terminal_a (str): Input Terminal A
        terminal_b (str): Input Terminal B
        terminal_z (str): Input Terminal Z
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        units: Literal["Degrees", "Radians", "Ticks"] | None = None,
        decoding_type: Literal["X1", "X2", "X4", "Two Pulse"] | None = None,
        pulses_per_rev: int | None = None,
        initial_angle: float | None = None,
        z_index_enable: bool | None = None,
        z_index_val: float | None = None,
        z_index_phase: (
            Literal["A High B High", "A High B Low", "A Low B High", "A Low B Low"]
            | None
        ) = None,
        terminal_a: str | None = None,
        terminal_b: str | None = None,
        terminal_z: str | None = None,
        **kwargs: Any,
    ) -> None:
        """Initialize angular position channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Position Angular",
            **kwargs,
        )
        self.layout = console.layout
        layout = self.layout

        # Units
        if units is not None:
            layout.click_btn("Units")
            layout.select_from_dropdown(units)
            self.form_values["Units"] = units
        else:
            self.form_values["Units"] = layout.get_dropdown_value("Units")

        # Initial Angle
        if initial_angle is not None:
            layout.fill_input_field("Initial Angle", str(initial_angle))
            self.form_values["Initial Angle"] = str(initial_angle)
        else:
            self.form_values["Initial Angle"] = layout.get_input_field("Initial Angle")

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

        # Z Index Enable
        if z_index_enable is not None:
            current_state = layout.get_toggle("Z Index Enable")
            if current_state != z_index_enable:
                layout.click_checkbox("Z Index Enable")
            self.form_values["Z Index Enable"] = z_index_enable
        else:
            self.form_values["Z Index Enable"] = layout.get_toggle("Z Index Enable")

        # Z Index Value
        if z_index_val is not None:
            layout.fill_input_field("Value", str(z_index_val))
            self.form_values["Value"] = str(z_index_val)
        else:
            self.form_values["Value"] = layout.get_input_field("Value")

        # Z Index Phase
        if z_index_phase is not None:
            layout.click_btn("Phase")
            layout.select_from_dropdown(z_index_phase)
            self.form_values["Phase"] = z_index_phase
        else:
            self.form_values["Phase"] = layout.get_dropdown_value("Phase")

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

        # Input Terminal Z
        if terminal_z is not None:
            layout.click_btn("Input Terminal Z")
            layout.select_from_dropdown(terminal_z)
            self.form_values["Input Terminal Z"] = terminal_z
        else:
            self.form_values["Input Terminal Z"] = layout.get_dropdown_value(
                "Input Terminal Z"
            )
