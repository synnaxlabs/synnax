#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

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

    channels_button: Locator
    channels_pane: Locator
    channels_list: Locator

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console
        # Channels button - try multiple approaches to find it
        self.channels_button = self.page.locator(
            "button.console-main-nav__item"
        ).filter(has=self.page.locator("svg.pluto-icon--channel"))

        # Backup selector - try finding by keyboard shortcut text "C"
        self.channels_button_backup = self.page.locator(
            "button.console-main-nav__item:has-text('C')"
        )
        self.channels_pane = self.page.locator("text=Channels").first
        self.channels_list = self.page.locator("div[id^='channel:']")

    def show_channels(self) -> None:
        if not self.channels_pane.is_visible():
            self.channels_button.click(force=True, timeout=2000)

        self.channels_pane.first.wait_for(state="visible", timeout=500)
        self.page.wait_for_timeout(100)

    def hide_channels(self) -> None:
        if self.channels_pane.is_visible():
            self.channels_button.click(force=True, timeout=2000)

    def create(
        self,
        *,
        name: ChannelName,
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

        if is_index and data_type == DataType.UNKNOWN:
            data_type = DataType.TIMESTAMP
        exists, _ = self.existing_channel(name)
        if exists:
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
        self.hide_channels()
        return True

    def existing_channel(self, name: ChannelName) -> tuple[bool, list[ChannelName]]:
        """
        Checks if a channel with the given name exists
        :param name: The name of the channel to check.
        :returns: A tuple containing a boolean indicating whether the channel exists
        and a list of all channels.
        """
        all_channels = self.list_all()
        exists = name in all_channels
        return exists, all_channels

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
        exists, all_channels = self.existing_channel(old_name)
        if not exists:
            raise ValueError(f"Channel {old_name} does not exist in {all_channels}")
        new_exists, _ = self.existing_channel(new_name)
        if new_exists:
            raise ValueError(f"Channel {new_name} already exists")

        # Find the channel in the list and rename it
        self.show_channels()
        for item in self.channels_list.all():
            if item.is_visible():
                # Get the channel name from the <p> element inside the channel div
                channel_name_element = item.locator("p.pluto-text--editable")
                text = channel_name_element.inner_text().strip()
                if text == old_name:
                    # Right click on the channel item to get context menu
                    item.click(button="right")
                    rename_option = self.page.get_by_text("Rename", exact=True).first
                    rename_option.click()

                    # The <p> element should now be editable - clear it and type new name
                    channel_name_element.click()
                    channel_name_element.fill(new_name)
                    self.page.keyboard.press("Enter")
                    self.page.wait_for_timeout(200)
                    break
        self.hide_channels()

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
        exists, all_channels = self.existing_channel(name)
        if not exists:
            raise ValueError(f"Channel {name} does not exist in {all_channels}")

        # Find the channel in the list and delete it
        for item in self.channels_list.all():
            if item.is_visible():
                # Get the channel name from the <p> element inside the channel div
                channel_name_element = item.locator("p.pluto-text--editable")
                text = channel_name_element.inner_text().strip()
                if text == name:
                    # Right click option
                    item.click(button="right")
                    delete_option = self.page.locator("text=Delete").first
                    delete_option.click()

                    # Delete button in Modal
                    self.page.get_by_role(
                        "button", name="Delete", exact=True
                    ).first.click()

                    # Check for notifications and close them if there's an error
                    i = -1
                    for notification in self.console.check_for_notifications():
                        i += 1
                        message = notification.get("message", "")
                        description = notification.get("description", "")
                        if (message == "Failed to delete Channel") and (
                            name in description
                        ):
                            # Close the notification before raising the error
                            self.console.close_notification(i)
                            raise RuntimeError(f"{message} {name}, {description}")
                    break
        self.hide_channels()

    def list_all(self) -> list[ChannelName]:
        """Lists all channels via console UI."""
        self.show_channels()

        all_items = self.channels_list.all()
        channels = list[ChannelName]()
        for item in all_items:
            if item.is_visible():
                # Extract channel name from the <p> element inside the channel div
                channel_name_element = item.locator("p.pluto-text--editable")
                channel_name = channel_name_element.inner_text().strip()
                channels.append(channel_name)

        return channels
