#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import time
from test.console.schematic import Schematic


class Schematic_Edit_Value(Schematic):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:

        self._log_message("Checking default properties of schematic value")
        # Create a schematic value with default properties
        node_id = self.add_schematic_value(f"{self.name}_uptime")
        default_props = self.get_schematic_value_props(node_id)

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
        self.edit_schematic_value_props(
            node_id,
            channel_name=f"{self.name}_time",
            notation="scientific",
            precision="4",
            averaging_window="4",
            stale_color="#FF0000",
            stale_timeout="10",
        )
        expected_edited_props = {
            "channel": f"{self.name}_time",
            "notation": "scientific",
            "precision": 4,
            "averaging_window": 4,
            "stale_color": "#FF0000",  # pluto-warning-m1
            "stale_timeout": 10,
        }
        edited_props = self.get_schematic_value_props(node_id)
        assert (
            edited_props == expected_edited_props
        ), f"Props mismatch!\nActual: {edited_props}\nExpected: {expected_edited_props}"
