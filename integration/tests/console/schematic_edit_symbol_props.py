#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from console.case import ConsoleCase


class Schematic_Edit_Symbol_Props(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:

        self.console.schematic.new()
        self.test_value_props()
        self.test_button_props()
        self.log("Test Complete")

    def test_value_props(self) -> None:
        self.log("Testing value props")
        console = self.console

        self.log("Checking default properties of schematic value")
        value = console.schematic.create_value(f"{self.name}_uptime")
        default_props = value.get_properties()

        expected_default_props = {
            "channel": f"{self.name}_uptime",
            "notation": "standard",
            "precision": 2,
            "averaging_window": 1,
            "stale_color": "#C29D0A",  # pluto-warning-m1
            "stale_timeout": 5,
        }
        assert (
            default_props == expected_default_props
        ), f"Props mismatch!\nActual: {default_props}\nExpected: {expected_default_props}"

        self.log("Checking edited properties of schematic value")
        expected_edited_props = {
            "channel": f"{self.name}_time",
            "notation": "scientific",
            "precision": 4,
            "averaging_window": 4,
            "stale_color": "#FF0000",
            "stale_timeout": 10,
        }
        value.edit_properties(
            channel_name=f"{self.name}_time",
            notation="scientific",
            precision=4,
            averaging_window=4,
            stale_color="#FF0000",
            stale_timeout=10,
        )
        edited_props = value.get_properties()
        assert (
            edited_props == expected_edited_props
        ), f"Props mismatch!\nActual: {edited_props}\nExpected: {expected_edited_props}"
        value.delete()

        self.log("Checking new node with non-default properties")
        non_default_props = {
            "channel": f"{self.name}_state",
            "notation": "engineering",
            "precision": 7,
            "averaging_window": 3,
            "stale_color": "#00FF00",
            "stale_timeout": 15,
        }
        non_default_value = console.schematic.create_value(
            f"{self.name}_state",
            notation="engineering",
            precision=7,
            averaging_window=3,
            stale_color="#00FF00",
            stale_timeout=15,
        )
        actual_non_default_props = non_default_value.get_properties()
        assert (
            actual_non_default_props == non_default_props
        ), f"Props mismatch!\nActual: {actual_non_default_props}\nExpected: {non_default_props}"
        non_default_value.delete()

    def test_button_props(self) -> None:
        self.log("Testing button props")
        console = self.console

        self.log("Creating channels")
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

        self.log("Creating schematic button")
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
        assert (
            default_props == expected_default_props
        ), f"Props mismatch!\nActual: {default_props}\nExpected: {expected_default_props}"

        self.log("Editing properties of schematic button")
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
        assert (
            edited_props == expected_edited_props
        ), f"Props mismatch!\nActual: {edited_props}\nExpected: {expected_edited_props}"
        button.delete()

        self.log("Checking non-default properties of schematic button")
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
        assert (
            actual_non_default_props == non_default_props
        ), f"Props mismatch!\nActual: {actual_non_default_props}\nExpected: {non_default_props}"
        non_default_button.delete()
