#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import abstractmethod
from typing import TYPE_CHECKING, Any

import synnax as sy
from playwright.sync_api import Page

from console.task.channels.analog import Analog

from ..page import ConsolePage

if TYPE_CHECKING:
    from console.console import Console


class NITask(ConsolePage):
    """NI Task automation interface for managing analog channels."""

    channels: list[Analog]
    channels_by_name: list[str]
    task_name: str

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.channels = []
        self.channels_by_name = []

    @abstractmethod
    def add_channel(
        self,
        name: str,
        chan_type: str,
        device: str,
        dev_name: str | None = None,
        **kwargs: Any,
    ) -> Analog:
        """
        Add a channel to the task.

        Subclasses must implement this method to validate the channel type
        and instantiate the appropriate channel class.

        Args:
            name: Channel name
            chan_type: Channel type string for UI selection
            device: Device identifier
            dev_name: Optional device name
            **kwargs: Additional channel-specific configuration

        Returns:
            The created channel instance
        """
        ...

    def _add_channel_helper(
        self,
        name: str,
        device: str,
        dev_name: str | None,
        channel_class: type[Analog],
        **kwargs: Any,
    ) -> Analog:
        """
        Helper method for adding a channel with common UI automation logic.

        Args:
            name: Channel name
            device: Device identifier
            dev_name: Optional device name
            channel_class: Channel class to instantiate
            **kwargs: Additional channel-specific configuration

        Returns:
            The created channel instance
        """
        console = self.console

        # Add first channel or subsequent channels
        if len(self.channels) == 0:
            console.click("Add a channel")
        else:
            console.page.locator("header:has-text('Channels') .pluto-icon--add").click()

        # Click the channel in the list
        idx = len(self.channels)
        console.page.locator(".pluto-list__item").nth(idx).click()

        # Configure device
        console.click_btn("Device")
        console.select_from_dropdown(device)

        if dev_name is None:
            dev_name = name[:12]
        # Handle device creation modal if it appears
        # Notifications will block the modal.
        console.close_all_notifications()
        sy.sleep(0.3)  # Give modal time to appear
        if console.check_for_modal():
            console.close_all_notifications()
            sy.sleep(0.2)
            console.fill_input_field("Name", dev_name)
            console.click_btn("Next")
            sy.sleep(0.2)
            console.fill_input_field("Identifier", dev_name)
            console.click_btn("Save")
            sy.sleep(0.2)

        if console.check_for_modal():
            raise RuntimeError("Blocking modal is still open")

        # Create channel using provided class
        channel = channel_class(console=console, name=name, device=device, **kwargs)

        self.channels.append(channel)
        self.channels_by_name.append(name)
        return channel

    def assert_channel(self, name: str | list[str]) -> None:
        """
        Assert a channel form is set correctly

        Args:
            name: Channel name or list of channel names to assert

        Returns: None
        """
        console = self.console
        names = [name] if isinstance(name, str) else name

        for channel_name in names:
            idx = self.channels_by_name.index(channel_name)
            console.page.locator(".pluto-list__item").nth(idx).click()
            channel = self.channels[idx]
            sy.sleep(0.1)
            channel.assert_form()

    def set_parameters(
        self,
        task_name: str | None = None,
        data_saving: bool | None = None,
        auto_start: bool | None = None,
    ) -> None:
        """
        Set the parameters for the task.

        Args:
            sample_rate: The sample rate for the task.
            stream_rate: The stream rate for the task.
            data_saving: Whether to save data to the core.
            auto_start: Whether to start the task automatically.
        """
        console = self.console

        if task_name is not None:
            console.fill_input_field("Name", task_name)
            console.ENTER

        if data_saving is not None:
            if data_saving != console.get_toggle("Data Saving"):
                console.click_checkbox("Data Saving")

        if auto_start is not None:
            if auto_start != console.get_toggle("Auto Start"):
                console.click_checkbox("Auto Start")

    def configure(self) -> None:
        # Notifications will block the configure channel.
        # Another mitigation is to snap the task page left.
        self.console.close_all_notifications()
        self.console.page.get_by_role("button", name="Configure", exact=True).click(
            force=True
        )

    def run(self) -> None:
        sy.sleep(0.2)
        self.console.page.locator("button .pluto-icon--play").locator("..").click(
            force=True
        )
        sy.sleep(0.2)

    def status(self) -> dict[str, str]:
        """
        Get the current status information from the task status box.

        Returns:
            Dictionary containing:
                - text: The status message (e.g., "Task has not been configured")
                - level: The alert level (e.g., "disabled", "info", "success", "error")
                - name: Status field name
                - time: Timestamp if available
        """
        sy.sleep(0.2)
        status_element = self.console.page.locator(
            ".console-task-state p.pluto-status__text, .console-task-state p.pluto-text"
        ).first

        # status
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
