#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
import synnax as sy


class Schematic_Edit_Button_Props(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:
        console = self.console
        console.schematic.new()

        self._log_message("Creating channels")
        CHANNEL_NAME = "button_cmd"
        INDEX_NAME = "button_idx"

        console.channels.create(
            name=INDEX_NAME,
            is_index=True,
        )
        console.channels.create(
            name=CHANNEL_NAME,
            data_type=sy.DataType.UINT8,
            is_index=False,
            index=INDEX_NAME,
        )

        self._log_message("Creating schematic button")
        button = console.schematic.create_button(
            channel_name=CHANNEL_NAME,
        )
        default_props = button.get_properties()
        expected_default_props = {
            "channel": CHANNEL_NAME,
            "activation_delay": 0,
            "show_control_chip": True,
            "mode": "fire",
        }
        assert default_props == expected_default_props, f"Props mismatch!\nActual: {default_props}\nExpected: {expected_default_props}"


        self._log_message("Editing properties of schematic button")
        button.edit_properties(
            channel_name=CHANNEL_NAME,
            activation_delay=4.2,
            show_control_chip=False,
            mode="Momentary",
        )
        expected_edited_props = {
            "channel": CHANNEL_NAME,
            "activation_delay": 4.2,
            "show_control_chip": False,
            "mode": "momentary",
        }

        edited_props = button.get_properties()
        assert edited_props == expected_edited_props, f"Props mismatch!\nActual: {edited_props}\nExpected: {expected_edited_props}"


        self._log_message("Checking non-default properties of schematic button")
        non_default_props = {
            "channel": CHANNEL_NAME,
            "activation_delay": 2.3,
            "show_control_chip": True,
            "mode": "pulse",
        }
        non_default_button = console.schematic.create_button(
            channel_name=CHANNEL_NAME,
            activation_delay=2.3,
            show_control_chip=True,
            mode="pulse",
        )
        actual_non_default_props = non_default_button.get_properties()
        assert actual_non_default_props == non_default_props, f"Props mismatch!\nActual: {actual_non_default_props}\nExpected: {non_default_props}"

        self._log_message("Test Complete")