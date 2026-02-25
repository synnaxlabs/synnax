#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from playwright.sync_api import Locator

from console.layout import LayoutClient
from console.page import ConsolePage


class TaskPage(ConsolePage):
    """Base class for task pages with common task operations.

    Provides common functionality for all task types (NI, LabJack, OPC UA, Modbus):
    - configure() - Configure the task
    - run() - Start the task
    - status() - Get task status
    - set_parameters() - Set common task parameters

    Subclasses should implement device-specific functionality like channel management.
    """

    pluto_label: str = ".console-task-configure"

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        *,
        pane_locator: Locator,
    ) -> None:
        """Initialize a TaskPage wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(layout, client, page_name, pane_locator=pane_locator)

    def configure(self) -> None:
        """Configure the task by clicking the Configure button."""
        configure_button = self.page.get_by_role("button", name="Configure", exact=True)
        configure_button.click(force=True)
        configure_button.wait_for(state="hidden")

    def run(self) -> None:
        """Start the task by clicking the play button."""
        play_button = self.page.locator("button .pluto-icon--play").locator("..")
        play_button.wait_for(state="visible", timeout=3000)
        play_button.click(timeout=5000)
        play_button.wait_for(state="hidden")

    def status(self) -> dict[str, str]:
        """Get the current status information from the task status box.

        Returns:
            Dictionary containing:
                - msg: The status message (e.g., "Task has not been configured")
                - level: The alert level (e.g., "disabled", "info", "success", "error")
        """
        status_element = self.page.locator(
            ".console-task-state p.pluto-status__text, .console-task-state p.pluto-text"
        ).first

        # Parse status level from CSS class
        class_attr = status_element.get_attribute("class") or ""
        level = "unknown"
        for cls in class_attr.split():
            if cls.startswith("pluto--status-"):
                level = cls.replace("pluto--status-", "")
                break

        msg = status_element.inner_text()

        return {
            "msg": msg,
            "level": level,
        }

    def copy_link(self) -> str:
        """Copy link to the task via the utility button in the form header."""
        link_button = self.page.locator(".pluto-icon--link").locator("..")
        link_button.click(timeout=5000)
        return self.layout.read_clipboard()

    def verify_config(self, expected_channels: list[str]) -> None:
        """Verify the task config page is visible and contains expected channels.

        Args:
            expected_channels: Channel identifiers expected in the channel list.
        """
        if self.pane_locator is None:
            raise RuntimeError("No pane locator available for config verification")
        self.pane_locator.wait_for(state="visible", timeout=10000)
        for channel in expected_channels:
            self.pane_locator.get_by_text(channel).first.wait_for(
                state="visible", timeout=5000
            )

    def set_parameters(
        self,
        *,
        task_name: str | None = None,
        data_saving: bool | None = None,
        auto_start: bool | None = None,
    ) -> None:
        """Set common task parameters.

        Args:
            task_name: The name for the task
            data_saving: Whether to save data to the server
            auto_start: Whether to start the task automatically
        """
        layout = self.layout

        if task_name is not None:
            layout.fill_input_field("Name", task_name)
            layout.press_enter()

        if data_saving is not None:
            if data_saving != layout.get_toggle("Data Saving"):
                layout.click_checkbox("Data Saving")

        if auto_start is not None:
            if auto_start != layout.get_toggle("Auto Start"):
                layout.click_checkbox("Auto Start")
