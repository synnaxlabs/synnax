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
        # Use the new node class approach
        """
        value_node = self.add_to_schematic("Value", f"{self.name}_uptime")
        print("Clicked on value by adding to schematic\n")

        value_node_1 = self.add_to_schematic("Value", f"{self.name}_uptime")
        print("Clicked on value by adding to schematic\n")

        
        
        value_node_2 = self.add_to_schematic("Value", f"{self.name}_time")
        print("Clicked on value by adding to schematic\n")

        setpoint_node = self.add_to_schematic("Setpoint", f"{self.name}_uptime")
        print('added setpoint node\n')

        setpoint_node = self.add_to_schematic("Setpoint", f"{self.name}_uptime")
        print('added setpoint node\n')
        """
        setpoint_node_1 = self.add_to_schematic("Setpoint", f"{self.name}_uptime")
        print('added setpoint node\n')

        setpoint_node_2 = self.add_to_schematic("Setpoint", f"{self.name}_uptime")
        print('added setpoint node\n')

        setpoint_node_1.set_control_authority("setpoint_in_the_label")

        setpoint_node_3 = self.add_to_schematic("Setpoint", f"{self.name}_uptime")
        

        print("done\n")
        time.sleep(10)