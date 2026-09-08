#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.task.opcua import OPCUAWrite
from tests.console.task.opcua_case import COMMAND_NODES, OPCUACase

NODE_IDS = list(COMMAND_NODES.values())


class OPCUAWriteTask(OPCUACase):
    """Configure, start, and stop an OPC UA write task through the Console form."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_channels(page)
        self.test_deploy(page)
        self.test_send_commands(page)
        self.test_stop(page)
        self.test_reopen_config(page, NODE_IDS)

    def test_create_task(self) -> OPCUAWrite:
        """Create the task page, select the server, and open its Browser."""
        self.log("Testing: Create OPC UA write task")
        page = self.create_page(OPCUAWrite, "OPCUA Console Write")
        self.select_server(page)
        return page

    def test_add_channels(self, page: OPCUAWrite) -> None:
        """Command variables dragged from the Browser are listed."""
        self.log("Testing: Add command channels from the Browser")
        page.add_channels(list(COMMAND_NODES))
        page.verify_config(NODE_IDS)

    def test_send_commands(self, page: OPCUAWrite) -> None:
        """Deploy created one command channel per node; commands reach the server."""
        self.log("Testing: Send commands")
        task = self.retrieve_task(page.page_name)
        node_ids = self.node_ids(task)
        assert node_ids == set(NODE_IDS), (
            f"Task node IDs should be {set(NODE_IDS)}, got {node_ids}"
        )
        keys = self.channel_keys(task)
        assert len(keys) == len(COMMAND_NODES), (
            f"Every command channel needs a key, got {keys}"
        )
        self.send_commands(page, task)
