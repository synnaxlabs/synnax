#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.task.opcua import OPCUARead
from tests.console.task.opcua_case import FLOAT_NODES, OPCUACase

NODE_IDS = list(FLOAT_NODES.values())


class OPCUAReadTask(OPCUACase):
    """Configure, start, and stop an OPC UA read task through the Console form."""

    def run(self) -> None:
        page = self.test_create_task()
        self.test_add_channels(page)
        self.test_set_rates(page)
        self.test_deploy(page)
        self.test_data_flow(page)
        self.test_reopen_config(page, NODE_IDS)

    def test_create_task(self) -> OPCUARead:
        """A new task is not deployed; select the server and open its Browser."""
        self.log("Testing: Create OPC UA read task")
        page = self.create_page(OPCUARead, "OPCUA Console Read")
        self.assert_not_deployed(page, "new task")
        self.select_server(page)
        return page

    def test_add_channels(self, page: OPCUARead) -> None:
        """Variables dragged from the Browser are listed with their node IDs."""
        self.log("Testing: Add channels from the Browser")
        page.add_channels(list(FLOAT_NODES))
        page.verify_config(NODE_IDS)

    def test_data_flow(self, page: OPCUARead) -> None:
        """Deploy created one channel per node; each collects samples at the rate."""
        self.log("Testing: Data flows on the created channels")
        task = self.retrieve_task(page.page_name)
        node_ids = self.node_ids(task)
        assert node_ids == set(NODE_IDS), (
            f"Task node IDs should be {set(NODE_IDS)}, got {node_ids}"
        )
        keys = self.channel_keys(task)
        assert len(keys) == len(FLOAT_NODES), f"Every channel needs a key, got {keys}"
        self.assert_sample_count(page, keys)
