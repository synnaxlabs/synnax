#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""OPC UA task pages for Console UI automation."""

import re

from playwright.sync_api import Locator, expect

from console.task_page import TaskPage

SERVER_FIELD_LABEL = "OPC UA server"
BROWSER_SELECTOR = ".console-opc-browser"
BROWSER_LOADING_SELECTOR = ".console-opc-browser__loading-icon"


class OPCUATask(TaskPage):
    """Shared OPC UA task page. Channels are added by dragging Browser variables."""

    def _browser(self) -> Locator:
        return self._pane().locator(BROWSER_SELECTOR).first

    def _node(self, name: str) -> Locator:
        pattern = re.compile(rf"^{re.escape(name)}$")
        return self._browser().get_by_role("treeitem").filter(has_text=pattern)

    def _channel_row(self, node_name: str) -> Locator:
        return self._channel_list().get_by_role("option").filter(has_text=node_name)

    def select_server(self, name: str) -> None:
        """Pick the OPC UA server the task talks to and wait for its Browser.

        :param name: Name of the server device.
        """
        # The option's accessible name is the device name joined to its endpoint.
        self.select_device(SERVER_FIELD_LABEL, name)
        browser = self._browser()
        browser.wait_for(state="visible", timeout=10000)
        browser.locator(BROWSER_LOADING_SELECTOR).wait_for(
            state="hidden", timeout=15000
        )

    def expand_node(self, name: str) -> None:
        """Expand a Browser node. Its children load through the rack's scan task.

        :param name: Display name of the node.
        """
        node = self._node(name)
        node.wait_for(state="visible", timeout=15000)
        if node.get_attribute("aria-expanded") == "true":
            return
        node.click()
        expect(node).to_have_attribute("aria-expanded", "true", timeout=5000)

    def add_channels(self, node_names: list[str]) -> None:
        """Drag Browser variables onto the channel list.

        :param node_names: Display names of the variable nodes to add.
        """
        channel_list = self._channel_list()
        channel_list.wait_for(state="visible", timeout=5000)
        for name in node_names:
            node = self._node(name)
            node.wait_for(state="visible", timeout=15000)
            node.drag_to(channel_list)
            self._channel_row(name).first.wait_for(state="visible", timeout=5000)

    def disable_channel(self, node_name: str) -> None:
        """Disable a listed channel via its context menu."""
        self.ctx_menu.action(self._channel_row(node_name).first, "Disable")

    def enable_channel(self, node_name: str) -> None:
        """Enable a listed channel via its context menu."""
        self.ctx_menu.action(self._channel_row(node_name).first, "Enable")


class OPCUARead(OPCUATask):
    """OPC UA read task page."""

    page_type = "OPC UA read task"
    pluto_label: str = ".console-task-configure--opc_read"


class OPCUAWrite(OPCUATask):
    """OPC UA write task page."""

    page_type = "OPC UA write task"
    pluto_label: str = ".console-task-configure--opc_write"
