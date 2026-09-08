#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.opcua import OPCUASim

import synnax as sy
from console.task.opcua import OPCUATask
from tests.console.task.task_case import ConsoleTaskCase

# Node name to node ID on the simulator. Float index 2 carries error injection.
FLOAT_NODES = {"my_float_0": "NS=2;I=8", "my_float_1": "NS=2;I=9"}
COMMAND_NODES = {f"command_{i}": f"NS=2;I={18 + i}" for i in range(3)}
BROWSER_ROOT_NODE = "MyObject"


class OPCUACase(ConsoleTaskCase):
    """ConsoleTaskCase against the OPC UA simulator."""

    sim_classes = [OPCUASim]

    def select_server(self, page: OPCUATask) -> None:
        """Select the simulator's server and open its object in the Browser."""
        page.select_server(self.device_name)
        page.expand_node(BROWSER_ROOT_NODE)

    @staticmethod
    def node_ids(task: sy.Task) -> set[str]:
        """Return the node IDs a task config points at."""
        channels = task.config.get("channels", [])
        return {ch.get("node_id", ch.get("nodeId")) for ch in channels}
