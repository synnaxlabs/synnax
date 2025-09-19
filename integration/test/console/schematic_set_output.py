#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from test.console.schematic import Schematic


class Schematic_Set_Output(Schematic):
    """
    Add a value component and edit its properties
    """

    def run(self) -> None:

        setpoint_node = self.add_to_schematic("Setpoint", f"{self.name}_uptime")
        setpoint_node.move(-200, 0)

        value_node = self.add_to_schematic("Value", f"{self.name}_uptime")
        value_node.move(200, 0)

        self.connect_nodes(setpoint_node, "right", value_node, "left")
        self.connect_nodes(value_node, "right", setpoint_node, "left")
        self.connect_nodes(setpoint_node, "bottom", value_node, "bottom")
        self.connect_nodes(value_node, "bottom", value_node, "right")

        self._log_message("Remove the time.sleep(10) before merge!!!")
        time.sleep(10)
