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
import time
from synnax.channel.payload import (
    ChannelKey,
    ChannelName,
    ChannelNames,
    normalize_channel_params,
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
        channels: sy.Channel | list[sy.Channel] | ChannelName | None = None,
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
        elif isinstance(channels, str):
            # Create from channel name string
            _name = channels
            _data_type = data_type
            _is_index = is_index
            _index = index
            _virtual = virtual
        else:
            # Overload 3: Create from list of Channel objects or strings
            # For simplicity, create the first channel in the list
            if len(channels) > 0:
                first_channel = channels[0]
                if isinstance(first_channel, str):
                    _name = first_channel
                    _data_type = data_type
                    _is_index = is_index
                    _index = index
                    _virtual = virtual
                else:
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
        self.hide_resources()
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
        time.sleep(0.1)
        return True

    def existing_channel(self, name: ChannelName) -> bool:
        """Checks if a channel with the given name exists"""
        channels = self.list_all()
        if name in channels:
            return True
        else:
            return False

    @overload
    def rename(self, name: ChannelName, new_name: ChannelName) -> bool:
        """Renames a single channel.

        :param name: The name of the channel to rename.
        :param new_name: The new name for the channel.
        :returns: True if successful, False otherwise.
        """
        ...

    @overload
    def rename(self, names: ChannelNames, new_names: ChannelNames) -> bool:
        """Renames multiple channels.

        :param names: The names of the channels to rename.
        :param new_names: The new names for the channels.
        :returns: True if successful, False otherwise.
        """
        ...

    def rename(self, names: ChannelNames, new_names: ChannelNames) -> bool:
        """Renames one or more channels via console UI.

        :param names: The name(s) of the channel(s) to rename.
        :param new_names: The new name(s) for the channel(s).
        :returns: True if successful, False otherwise.
        """
        try:
            # Normalize inputs to lists
            normalized_names = normalize_channel_params(names)
            normalized_new_names = normalize_channel_params(new_names)

            # Ensure we have the same number of names and new names
            if len(normalized_names.channels) != len(normalized_new_names.channels):
                return False

            # Rename each channel via console UI
            for old_name, new_name in zip(normalized_names.channels, normalized_new_names.channels):
                self._rename_single_channel(str(old_name), str(new_name))

            return True
        except Exception:
            return False

    def _rename_single_channel(self, old_name: str, new_name: str) -> None:
        """Renames a single channel via console UI."""
        self.show_channels()

        if not self.existing_channel(old_name):
            raise ValueError(f"Channel {old_name} does not exist")
        if self.existing_channel(new_name):
            raise ValueError(f"Channel {new_name} already exists")

        # Find the channel in the list and right-click it
        for item in self.channels_list.all():
            if item.is_visible():
                text = item.inner_text().strip()
                if text == old_name:
                    # Right click option
                    item.click(button="right")
                    rename_option = self.page.get_by_text("Rename", exact=True).first
                    rename_option.click()
                    # Double click to edit the name directly
                    item.dblclick()
                    self.page.keyboard.type(new_name)
                    self.page.keyboard.press("Enter")
                    time.sleep(0.1)
                    break

    @overload
    def delete(self, name: ChannelName) -> None:
        """Deletes a single channel.

        :param name: The name of the channel to delete.
        :returns: None.
        """
        ...

    @overload
    def delete(self, names: ChannelNames) -> None:
        """Deletes multiple channels.

        :param names: The names of the channels to delete.
        :returns: None.
        """
        ...

    def delete(self, names: ChannelNames) -> None:
        """Deletes one or more channels via console UI.

        :param names: The name(s) of the channel(s) to delete.
        :returns: None.
        """
        # Normalize inputs to lists
        normalized_names = normalize_channel_params(names)

        # Ensure we have the same number of names and new names
        if len(normalized_names.channels) != len(normalized_names.channels):
            raise ValueError("Number of names and new names must be equal")

        # Delete each channel via console UI
        for name in normalized_names.channels:
            self._delete_single_channel(str(name))

    def _delete_single_channel(self, name: str) -> None:
        """Deletes a single channel via console UI."""
        self.show_channels()

        if not self.existing_channel(name):
            raise ValueError(f"Channel {name} does not exist")  

        # Find the channel in the list and right-click it
        for item in self.channels_list.all():
            if item.is_visible():
                text = item.inner_text().strip()
                if text == name:

                    # Right click option
                    item.click(button="right")
                    delete_option = self.page.locator("text=Delete").first
                    delete_option.click()

                    # Delete Modal button
                    self.page.get_by_role("button", name="Delete", exact=True).click()
                    time.sleep(0.1)
                    break

    def list_all(self) -> list[ChannelName]:
        """Lists all channels via console UI."""
        self.show_channels()
        channels = list[ChannelName]()
        for item in self.channels_list.all():
            if item.is_visible():
                channels.append(item.inner_text().strip())
        return channels