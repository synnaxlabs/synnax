#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import abstractmethod
from typing import Any, TypeVar

from playwright.sync_api import Locator

import synnax as sy
from console.devices import fill_configure
from console.layout import LayoutClient
from console.task.channels.analog import Analog
from console.task.channels.counter import Counter
from console.task_page import TaskPage

# Union type for all NI channel types
NIChannel = Analog | Counter
_NIChannelT = TypeVar("_NIChannelT", bound=NIChannel)


class NITask(TaskPage):
    """NI Task automation interface for managing channels."""

    channels: list[NIChannel]
    channels_by_name: list[str]
    task_name: str

    def __init__(
        self,
        *,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        pane_locator: Locator,
    ) -> None:
        """Initialize an NITask page wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(
            layout=layout, client=client, page_name=page_name, pane_locator=pane_locator
        )
        self.channels: list[NIChannel] = []
        self.channels_by_name: list[str] = []

    @abstractmethod
    def add_channel(
        self,
        name: str,
        *,
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

    def _open_new_device_dropdown(self) -> None:
        """Select the next channel row and open its device dropdown."""
        idx = self.next_channel_row()
        self.layout.page.locator(".lyra-list__item").nth(idx).click()
        self.layout.click_btn("Device")

    def connect_device(self) -> None:
        """Open the find view from the "Connect NI hardware" action of the device
        dropdown of the next channel row. The device configured there becomes the
        device of the row. The page does not track the row as a channel."""
        self._open_new_device_dropdown()
        self.layout.dialog.get_by_text("Connect NI hardware", exact=True).click(
            timeout=5000
        )
        self.layout.page.locator(self.layout.MODAL_SELECTOR).wait_for(
            state="visible", timeout=5000
        )

    def _add_channel_helper(
        self,
        name: str,
        device: str,
        dev_name: str | None,
        channel_class: type[_NIChannelT],
        **kwargs: Any,
    ) -> _NIChannelT:
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

        self._open_new_device_dropdown()
        layout.select_from_dropdown(device)

        if dev_name is None:
            dev_name = name[:12]
        # Handle device creation modal if it appears
        sy.sleep(0.2)  # Give modal time to appear
        configured = layout.is_modal_open()
        if configured:
            modal = layout.page.locator(layout.MODAL_SELECTOR)
            fill_configure(
                modal,
                notifications=layout.notifications,
                name=dev_name,
                identifier=dev_name,
            )
            modal.wait_for(state="hidden", timeout=5000)
            # Configuring renames the device, and the field shows the new name late.
            layout.wait_for_dropdown_value("Device", dev_name)

        # Create channel using provided class
        channel = channel_class(layout=self.layout, name=name, device=device, **kwargs)

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
        names = [name] if isinstance(name, str) else name

        for channel_name in names:
            idx = self.channels_by_name.index(channel_name)
            self.layout.page.locator(".lyra-list__item").nth(idx).click()
            channel = self.channels[idx]
            sy.sleep(0.3)
            channel.assert_form()
