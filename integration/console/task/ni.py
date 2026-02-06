#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import abstractmethod
from typing import Any, TypeVar, cast

import synnax as sy
from playwright.sync_api import Locator

from console.layout import LayoutClient
from console.task.channels.analog import Analog
from console.task.channels.counter import Counter
from console.task_page import TaskPage

# Union type for all NI channel types
NIChannel = Analog | Counter
NIChannelT = TypeVar("NIChannelT", bound=NIChannel)


class NITask(TaskPage):
    """NI Task automation interface for managing channels."""

    channels: list[NIChannel]
    channels_by_name: list[str]
    task_name: str

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        *,
        pane_locator: Locator,
    ) -> None:
        """Initialize an NITask page wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(layout, client, page_name, pane_locator=pane_locator)
        self.channels: list[NIChannel] = []
        self.channels_by_name: list[str] = []

    @abstractmethod
    def add_channel(
        self,
        name: str,
        chan_type: str,
        device: str,
        dev_name: str | None = None,
        **kwargs: Any,
    ) -> NIChannel:
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
        channel_class: type[NIChannelT],
        **kwargs: Any,
    ) -> NIChannelT:
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
        layout = self.layout

        # Add first channel or subsequent channels
        if len(self.channels) == 0:
            layout.click("Add a channel")
        else:
            layout.page.locator("header:has-text('Channels') .pluto-icon--add").click()

        # Click the channel in the list
        idx = len(self.channels)
        layout.page.locator(".pluto-list__item").nth(idx).click()

        # Configure device
        layout.click_btn("Device")
        layout.select_from_dropdown(device)

        if dev_name is None:
            dev_name = name[:12]
        # Handle device creation modal if it appears
        sy.sleep(0.2)  # Give modal time to appear
        if layout.is_modal_open():
            sy.sleep(0.2)
            layout.fill_input_field("Name", dev_name)
            layout.click_btn("Next")
            sy.sleep(0.2)
            layout.fill_input_field("Identifier", dev_name)
            layout.click_btn("Save")
            sy.sleep(0.2)

        if layout.is_modal_open():
            raise RuntimeError("Blocking modal is still open")

        # Create channel using provided class
        channel = channel_class(layout=self.layout, name=name, device=device, **kwargs)

        self.channels.append(channel)
        self.channels_by_name.append(name)
        return cast(NIChannelT, channel)

    def assert_channel(self, name: str | list[str]) -> None:
        """
        Assert a channel form is set correctly

        Args:
            name: Channel name or list of channel names to assert

        Returns: None
        """
        names = [name] if isinstance(name, str) else name

        for channel_name in names:
            idx = self.channels_by_name.index(channel_name)
            self.layout.page.locator(".pluto-list__item").nth(idx).click()
            channel = self.channels[idx]
            sy.sleep(0.1)
            channel.assert_form()
