#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from typing import TYPE_CHECKING

from playwright.sync_api import Locator, Page
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
        self.resources_button = self.page.locator(
            "button.console-main-nav__item"
        ).filter(has=self.page.locator("svg.pluto-icon--group"))
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

    def create(
        self,
        name: ChannelName | None = None,
        *,
        data_type: CrudeDataType = DataType.UNKNOWN,
        is_index: bool = False,
        index: ChannelKey = 0,
        virtual: bool = False,
    ) -> bool:
        """Creates a new channel via console UI.

        :param name: The name for the channel.
        :param data_type: The data type of the samples in the channel. For example, `"float32"`.
        :param is_index: Boolean indicating whether the channel is an index. Index
        channels should have a data type of TIMESTAMP.
        :param index: The key of the channel that indexes this channel.
        :param virtual: Boolean indicating whether the channel is virtual. Virtual
        channels do not store any data, and are used for streaming purposes only.
        :returns: True if the channel was created successfully.
        """
        if name is None:
            return False

        if is_index and data_type == DataType.UNKNOWN:
            data_type = DataType.TIMESTAMP

        if self.existing_channel(name):
            self.hide_resources()
            return False

        # Open command palette and create channel
        self.console.command_palette("Create a Channel")

        # Fill channel name
        self.console.fill_input_field("Name", name)

        # Set virtual if needed
        if virtual:
            self.console.click_checkbox("Virtual")

        # Configure as index or regular channel
        if is_index:
            self.console.click_checkbox("Is Index")
        else:
            if index == 0:
                raise ValueError("Index must be provided if is_index is False")

            # Set data type
            data_type_str = str(DataType(data_type))
            self.console.click_btn("Data Type")
            self.console.select_from_dropdown(data_type_str, "Search Data Types")

            # Set index - index should be the channel name
            self.console.click_btn("Index")
            self.console.select_from_dropdown(index, "Search Channels")

        # Select "Create" button
        self.page.get_by_role("button", name="Create", exact=True).click()
        self.hide_resources()
        return True

    def existing_channel(self, name: ChannelName) -> bool:
        """Checks if a channel with the given name exists"""
        return name in self.list_all()

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
            for old_name, new_name in zip(
                normalized_names.channels, normalized_new_names.channels
            ):
                self._rename_single_channel(str(old_name), str(new_name))

            return True
        except Exception:
            return False

    def _rename_single_channel(self, old_name: str, new_name: str) -> None:
        """Renames a single channel via console UI."""
        if not self.existing_channel(old_name):
            raise ValueError(f"Channel {old_name} does not exist")
        if self.existing_channel(new_name):
            raise ValueError(f"Channel {new_name} already exists")

        # Find the channel in the list and right-click it
        self.show_channels()
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
                    break
        self.hide_resources()

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
        if not self.existing_channel(name):
            raise ValueError(f"Channel {name} does not exist")

        # Find the channel in the list and right-click it
        self.show_channels()
        for item in self.channels_list.all():
            if item.is_visible():
                text = item.inner_text().strip()
                if text == name:

                    # Right click option
                    item.click(button="right")
                    delete_option = self.page.locator("text=Delete").first
                    delete_option.click()

                    # Delete button in Modal
                    self.page.get_by_role("button", name="Delete", exact=True).click()
                    break
        self.hide_resources()

    def list_all(self) -> list[ChannelName]:
        """Lists all channels via console UI."""
        self.show_channels()
        channels = list[ChannelName]()
        for item in self.channels_list.all():
            if item.is_visible():
                channels.append(item.inner_text().strip())
        return channels
