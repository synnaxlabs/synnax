#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""OPC UA invalid configuration tests through the Console.

The Console form blocks some invalid settings before they reach the Driver and
renders a field error; the rest deploy and the Driver rejects them with an error
status. The driver-side cases that need a hand-edited config (bad channel key, bad
node ID) cannot be reached through the form and stay with ``driver/opcua_invalid``.
"""

from console.task.opcua import SERVER_FIELD_LABEL, OPCUARead
from tests.console.task.opcua_case import FLOAT_NODES, OPCUACase

NODE = next(iter(FLOAT_NODES))


class OPCUAInvalidConfig(OPCUACase):
    """Verify invalid OPC UA read task configurations are rejected.

    Tests (run sequentially on one task page):
        1. No device selected: the form blocks the deploy.
        2. Invalid rates: stream rate above sample rate, the form blocks the deploy.
        3. No enabled channels: the Driver rejects the start.
        4. Duplicate channel: a second task on the same channel is rejected while
           the first one runs.
    """

    def run(self) -> None:
        page = self.test_no_device()
        self.test_invalid_rates(page)
        self.test_no_enabled_channels(page)
        self.test_duplicate_channel(page)

    def test_no_device(self) -> OPCUARead:
        self.log("Testing: No device selected")
        page = self.create_page(OPCUARead, "OPCUA Console Invalid")
        self.assert_device_required(page, SERVER_FIELD_LABEL)
        return page

    def test_invalid_rates(self, page: OPCUARead) -> None:
        self.log("Testing: Invalid rates (sample_rate < stream_rate)")
        self.select_server(page)
        page.add_channels([NODE])
        self.assert_rates_rejected(page)

    def test_no_enabled_channels(self, page: OPCUARead) -> None:
        self.log("Testing: No enabled channels")
        self.set_rates(page)
        page.disable_channel(NODE)
        self.assert_driver_rejects(page, "no enabled channels")

    def test_duplicate_channel(self, page: OPCUARead) -> None:
        """Run the task, then deploy a second task that reads the same node."""
        self.log("Testing: Duplicate channel (two tasks on same channel)")
        page.enable_channel(NODE)
        page.deploy()
        # Page objects locate their pane by task type, so the first tab closes before a
        # second read task opens.
        self.console.close_all_tabs()
        second = self.create_page(OPCUARead, "OPCUA Console Duplicate")
        self.select_server(second)
        second.add_channels([NODE])
        self.assert_driver_rejects(second, "duplicate channel")
        self.console.tasks.stop(page.page_name)
