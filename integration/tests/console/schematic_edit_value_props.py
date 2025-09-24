#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from test.console.case import ConsoleCase


class Schematic_Edit_Value_Props(ConsoleCase):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:
        console = self.console
        console.schematic.new()

        self._log_message("Checking default properties of schematic value")
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

        self._log_message("Checking edited properties of schematic value")
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

        self._log_message("Checking new node with non-default properties")
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

        console.schematic.screenshot()
