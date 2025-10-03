#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Literal, Optional

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


class FunctionGenerator(Analog):
    """
    Function Generator channel type for NI analog output tasks.

    Kwargs:
        port (int): Physical port number
        frequency (float): Frequency in Hz
        amplitude (float): Amplitude in V
        offset (float): Offset in V
        waveform (str): "Sine", "Triangle", "Square", "Sawtooth"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        port: Optional[int] = None,
        frequency: Optional[float] = None,
        amplitude: Optional[float] = None,
        offset: Optional[float] = None,
        waveform: Optional[Literal["Sine", "Triangle", "Square", "Sawtooth"]] = None,
    ) -> None:
        """Initialize function generator channel with configuration."""

        # Does not call super()

        self.console = console
        self.device = device
        self.name = name

        values = {}

        # Configure channel type
        console.click_btn("Channel Type")
        console.select_from_dropdown("Function Generator")
        values["Channel Type"] = "Function Generator"

        # Get device (set by task.add_channel)
        values["Device"] = console.get_dropdown_value("Device")

        # Optional configurations
        if port is not None:
            console.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = console.get_input_field("Port")

        # Function generator-specific configurations
        if frequency is not None:
            console.fill_input_field("Frequency", str(frequency))
            values["Frequency"] = str(frequency)
        else:
            values["Frequency"] = console.get_input_field("Frequency")

        if amplitude is not None:
            console.fill_input_field("Amplitude", str(amplitude))
            values["Amplitude"] = str(amplitude)
        else:
            values["Amplitude"] = console.get_input_field("Amplitude")

        if offset is not None:
            console.fill_input_field("Offset", str(offset))
            values["Offset"] = str(offset)
        else:
            values["Offset"] = console.get_input_field("Offset")

        if waveform is not None:
            console.page.get_by_text(waveform).click()
            values["Waveform"] = waveform
        else:
            # Determine which waveform button is currently selected
            waveform_options = ["Sine", "Triangle", "Square", "Sawtooth"]
            for option in waveform_options:
                try:
                    button = console.page.get_by_text(option).first
                    if button.count() > 0:
                        class_name = button.get_attribute("class") or ""
                        if "pluto-btn--filled" in class_name:
                            values["Waveform"] = option
                            break
                except:
                    continue

        self.form_values = values

    def assert_form(self) -> None:
        """Override assert_form to handle button selector for waveform."""
        for key, expected_value in self.form_values.items():
            # Handle waveform button selector (no label)
            if key == "Waveform":
                waveform_options = ["Sine", "Triangle", "Square", "Sawtooth"]
                actual_value = self.console.get_selected_button(waveform_options)
            else:
                # Handle normally
                try:
                    actual_value = self.console.get_input_field(key, timeout=100)
                except:
                    actual_value = self.console.get_dropdown_value(key)

                if actual_value.strip() == "":
                    actual_value = "0"

            assert (
                actual_value == expected_value
            ), f"Channel {self.name} Form value '{key}' - Expected: {expected_value} - Actual: {actual_value}"
