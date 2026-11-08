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

NAMESPACE_URI = "http://examples.freeopcua.github.io"
# Node name to node ID on the simulator, as the Browser gives it. Float index 2 carries
# error injection.
FLOAT_NODES = {f"my_float_{i}": f"nsu={NAMESPACE_URI};i={8 + i}" for i in range(2)}
COMMAND_NODES = {f"command_{i}": f"nsu={NAMESPACE_URI};i={18 + i}" for i in range(3)}
BROWSER_ROOT_NODE = "MyObject"


class OPCUACase(ConsoleTaskCase):
    """ConsoleTaskCase against the OPC UA simulator."""

    sim_classes = [OPCUASim]

    def select_server(self, page: OPCUATask) -> None:
        """Select the simulator's server and open its object in the Browser."""
        page.select_server(self.device_name)
        page.expand_node(BROWSER_ROOT_NODE)

    @classmethod
    def node_ids(cls, task: sy.Task) -> set[str]:
        """Return the node IDs a task config points at."""
        ids: set[str] = set()
        for ch in cls.config_channels(task):
            node_id = ch.get("node_id", ch.get("nodeId"))
            if not isinstance(node_id, str):
                raise TypeError(f"task config node ID {node_id!r} is not a string")
            ids.add(node_id)
        return ids
