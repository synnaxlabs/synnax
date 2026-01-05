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


class LinearPosition(Counter):
    """
    Linear Position channel type for NI counter read tasks.

    Kwargs:
        port (int): Physical port number
        units (str): "Meters", "Inches", or "Ticks"
        decoding_type (str): "X1", "X2", "X4", or "Two Pulse"
        dist_per_pulse (float): Distance per pulse
        initial_pos (float): Initial position value
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
        units: Literal["Meters", "Inches", "Ticks"] | None = None,
        decoding_type: Literal["X1", "X2", "X4", "Two Pulse"] | None = None,
        dist_per_pulse: float | None = None,
        initial_pos: float | None = None,
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
        """Initialize linear position channel with configuration."""
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Position Linear",
            **kwargs,
        )

        # Units
        if units is not None:
            console.click_btn("Units")
            console.select_from_dropdown(units)
            self.form_values["Units"] = units
        else:
            self.form_values["Units"] = console.get_dropdown_value("Units")

        # Initial Position
        if initial_pos is not None:
            console.fill_input_field("Initial Position", str(initial_pos))
            self.form_values["Initial Position"] = str(initial_pos)
        else:
            self.form_values["Initial Position"] = console.get_input_field(
                "Initial Position"
            )

        # Distance per Pulse
        if dist_per_pulse is not None:
            console.fill_input_field("Distance / Pulse", str(dist_per_pulse))
            self.form_values["Distance / Pulse"] = str(dist_per_pulse)
        else:
            self.form_values["Distance / Pulse"] = console.get_input_field(
                "Distance / Pulse"
            )

        # Decoding Type
        if decoding_type is not None:
            console.click_btn("Decoding Type")
            console.select_from_dropdown(decoding_type)
            self.form_values["Decoding Type"] = decoding_type
        else:
            self.form_values["Decoding Type"] = console.get_dropdown_value(
                "Decoding Type"
            )

        # Z Index Enable
        if z_index_enable is not None:
            current_state = console.get_toggle("Z Index Enable")
            if current_state != z_index_enable:
                console.click_checkbox("Z Index Enable")
            self.form_values["Z Index Enable"] = z_index_enable
        else:
            self.form_values["Z Index Enable"] = console.get_toggle("Z Index Enable")

        # Z Index Value
        if z_index_val is not None:
            console.fill_input_field("Value", str(z_index_val))
            self.form_values["Value"] = str(z_index_val)
        else:
            self.form_values["Value"] = console.get_input_field("Value")

        # Z Index Phase
        if z_index_phase is not None:
            console.click_btn("Phase")
            console.select_from_dropdown(z_index_phase)
            self.form_values["Phase"] = z_index_phase
        else:
            self.form_values["Phase"] = console.get_dropdown_value("Phase")

        # Input Terminal A
        if terminal_a is not None:
            console.click_btn("Input Terminal A")
            console.select_from_dropdown(terminal_a)
            self.form_values["Input Terminal A"] = terminal_a
        else:
            self.form_values["Input Terminal A"] = console.get_dropdown_value(
                "Input Terminal A"
            )

        # Input Terminal B
        if terminal_b is not None:
            console.click_btn("Input Terminal B")
            console.select_from_dropdown(terminal_b)
            self.form_values["Input Terminal B"] = terminal_b
        else:
            self.form_values["Input Terminal B"] = console.get_dropdown_value(
                "Input Terminal B"
            )

        # Input Terminal Z
        if terminal_z is not None:
            console.click_btn("Input Terminal Z")
            console.select_from_dropdown(terminal_z)
            self.form_values["Input Terminal Z"] = terminal_z
        else:
            self.form_values["Input Terminal Z"] = console.get_dropdown_value(
                "Input Terminal Z"
            )
