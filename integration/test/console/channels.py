#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, overload

import synnax as sy
from synnax.channel.payload import (
    ChannelKey,
    ChannelName,
)
from synnax.telem import (
    CrudeDataType,
    DataType,
)
from playwright.sync_api import Page, Locator

if TYPE_CHECKING:
    from .console import Console

class ChannelClient:
    """Console channel client"""

    resources_button: Locator
    resources_pane: Locator
    channels_dropdown: Locator
    channels_list: Locator

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console
        # Resource Pane
        self.resources_button = self.page.locator("button.console-main-nav__item").filter(has=self.page.locator("svg.pluto-icon--group"))
        self.resources_pane = self.page.locator("text=Resources").first
        
        # Channels section within resources pane
        self.channels_dropdown = self.page.locator("text=Channels").first
        self.channels_list = self.page.locator("div[id^='channel:']")
        
    def show_channels(self) -> None:
        self.show_resources()
        if self.channels_list.count() == 0:
            self.channels_dropdown.click(force=True)
        self.channels_list.first.wait_for(state="visible", timeout=500)

    def hide_channels(self) -> None:
        if self.channels_list.is_visible():
            self.channels_dropdown.click(force=True)

    def show_resources(self) -> None:
        if not self.resources_pane.is_visible():
            self.resources_button.click(force=True)
        self.resources_pane.first.wait_for(state="visible", timeout=500)

    def hide_resources(self) -> None:
        if self.resources_pane.is_visible():
            self.resources_button.click(force=True)

    @overload
    def create(self, channels: sy.Channel) -> bool: ...

    @overload
    def create(self, channels: list[sy.Channel]) -> bool: ...

    def create(
        self,
        channels: sy.Channel | list[sy.Channel] | None = None,
        *,
        data_type: CrudeDataType = DataType.UNKNOWN,
        name: ChannelName = "",
        is_index: bool = False,
        index: ChannelKey = 0,
        virtual: bool = False,
    ) -> bool:
        """Creates new channel(s) via console UI.

        :param channels: Optional single channel or list of channels to create.
        :param data_type: The data type of the samples in the channel. For example, `"float32"`.
        :param name: A name for the channel.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have a data type of sy.TIMESTAMP.
        :param index: The key of the channel that indexes this channel.
        :param virtual: Boolean indicating whether the channel is virtual. Virtual
        channels do not store any data, and are used for streaming purposes only.
        :returns: True if the channel(s) were created successfully.
        """
        # Handle different overload cases
        if channels is None:
            # Overload 1: Create from parameters
            if is_index and data_type == DataType.UNKNOWN:
                data_type = DataType.TIMESTAMP
            _name = name
            _data_type = data_type
            _is_index = is_index
            _index = index
            _virtual = virtual
        elif isinstance(channels, sy.Channel):
            # Overload 2: Create from single Channel object
            _name = channels.name
            _data_type = channels.data_type
            _is_index = channels.is_index
            _index = channels.index
            _virtual = channels.virtual
        else:
            # Overload 3: Create from list of Channel objects
            # For simplicity, create the first channel in the list
            if len(channels) > 0:
                first_channel = channels[0]
                _name = first_channel.name
                _data_type = first_channel.data_type
                _is_index = first_channel.is_index
                _index = first_channel.index
                _virtual = first_channel.virtual
            else:
                # No channels to create
                return True

        if self.existing_channel(_name):
            return False
    
        # Open command palette and create channel
        self.console.command_palette("Create a Channel")

        # Fill channel name
        name_field = self.page.locator("text=Name").locator("..").locator("input")
        name_field.fill(_name)

        # Set virtual if needed
        if _virtual:
            self.page.get_by_text("Virtual").click()

        # Configure as index or regular channel
        if _is_index:
            is_index_toggle = (
                self.page.locator("text=Is Index")
                .locator("..")
                .locator("input[type='checkbox']")
                .first
            )
            is_index_toggle.click()
        else:
            if _index == 0:
                raise ValueError("Index must be provided if is_index is False")

            # Set data type
            data_type_str = str(DataType(_data_type))
            self.console._select_from_dropdown("Data Type", data_type_str)

            # Set index - _index should be the channel name
            self.console._select_from_dropdown("Index", _index)

        # Select "Create" button
        self.page.get_by_role("button", name="Create", exact=True).click()
    
        return True

    def existing_channel(self, name: ChannelName) -> bool:
        """Checks if a channel with the given name exists"""
        self.show_channels()
        exists = False
        for item in self.channels_list.all():
            if item.is_visible():
                text = item.inner_text().strip()
                if text == name:
                    exists = True
                    break
        self.hide_resources()
        return exists