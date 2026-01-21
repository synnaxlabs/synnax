#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
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
        if self.channels_pane.is_visible():
            return
        self.channels_button.click(force=True, timeout=2000)
        self.channels_pane.first.wait_for(state="visible", timeout=500)
        self.channels_list.first.wait_for(state="attached", timeout=2000)

    def hide_channels(self) -> None:
        if self.channels_pane.is_visible():
            self.channels_button.click(force=True, timeout=2000)

    def _find_channel_item(self, name: ChannelName) -> Locator | None:
        """Find a channel item in the list by name.

        :param name: The channel name to find.
        :returns: The channel item Locator, or None if not found.
        """
        for item in self.channels_list.all():
            if not item.is_visible():
                continue
            channel_name_element = item.locator("p.pluto-text--editable")
            text = channel_name_element.inner_text().strip()
            if text == name:
                return item
        return None

    def _right_click_channel(self, name: ChannelName) -> Locator:
        """Find a channel and right-click it to open context menu.

        :param name: The channel name to right-click.
        :returns: The channel item Locator.
        :raises ValueError: If channel not found.
        """
        self.show_channels()
        item = self._find_channel_item(name)
        if item is None:
            raise ValueError(f"Channel {name} not found")
        item.click(button="right")
        return item

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

        self.open_create_modal()
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

    def create_with_create_more(
        self,
        channels: list[dict[str, str | int | bool]],
    ) -> list[ChannelName]:
        """Creates multiple channels using the 'Create More' checkbox.

        Each channel dict should contain:
            - name: The name for the channel (required)
            - data_type: The data type (optional, default uses index type)
            - is_index: Boolean for index channel (optional, default False)
            - index: The key of the index channel (required if not is_index)
            - virtual: Boolean for virtual channel (optional, default False)

        :param channels: List of channel configuration dicts
        :returns: List of created channel names
        """
        if not channels:
            return []

        created_channels: list[ChannelName] = []

        for i, ch_config in enumerate(channels):
            name = str(ch_config["name"])
            data_type = ch_config.get("data_type", DataType.UNKNOWN)
            is_index = ch_config.get("is_index", False)
            index = ch_config.get("index", 0)
            virtual = ch_config.get("virtual", False)
            index_str = str(index) if index != 0 else ""

            if is_index and data_type == DataType.UNKNOWN:
                data_type = DataType.TIMESTAMP

            # Check if channel already exists
            exists, _ = self.existing_channel(name)
            if exists:
                continue

            # Open command palette for first channel or if modal closed
            if i == 0:
                self.console.command_palette("Create a Channel")
                # Wait for modal to appear
                self.page.wait_for_selector(
                    "div.pluto-dialog__dialog.pluto--modal.pluto--visible",
                    timeout=5000,
                )

            # Fill channel name (use same pattern as create method)
            name_input = self.page.get_by_role("textbox", name="Name")
            name_input.fill(name)

            # Set virtual if needed
            if virtual:
                self.console.click_checkbox("Virtual")

            # Configure as index or regular channel
            if is_index:
                self.console.click_checkbox("Is Index")
            else:
                if index == 0:
                    raise ValueError(
                        f"Index must be provided for non-index channel: {name}"
                    )

                # Set data type
                data_type_str = str(DataType(data_type))
                self.console.click_btn("Data Type")
                self.console.select_from_dropdown(data_type_str, "Search Data Types")

                # Set index
                self.console.click_btn("Index")
                self.console.select_from_dropdown(index_str, "Search Channels")

            # Check "Create More" for all but the last channel
            is_last = i == len(channels) - 1
            create_more_checkbox = (
                self.page.locator("text=Create More")
                .locator("..")
                .locator("input[type='checkbox']")
                .first
            )

            if not is_last:
                # Ensure "Create More" is checked
                if not create_more_checkbox.is_checked():
                    create_more_checkbox.click()
            else:
                # Ensure "Create More" is unchecked for last channel
                if create_more_checkbox.is_checked():
                    create_more_checkbox.click()

            self.page.get_by_role("button", name="Create", exact=True).click()
            self.page.wait_for_timeout(200)  # Wait for channel creation
            created_channels.append(name)

        self.hide_channels()
        return created_channels

    def create_calculated(self, *, name: ChannelName, expression: str) -> str | None:
        """Creates a calculated channel via console UI.

        :param name: The name for the calculated channel.
        :param expression: The calculation expression (e.g., "channel_a * 2").
        :returns: None if successful, error message string if failed.
        """
        exists, _ = self.existing_channel(name)
        if exists:
            return "Channel already exists"

        self.open_create_calculated_modal()

        name_input = self.page.locator("input[placeholder='Name']")
        name_input.fill(name)

        editor = self.page.locator(".monaco-editor")
        editor.click()
        self.page.locator(".monaco-editor.focused").wait_for(
            state="visible", timeout=2000
        )
        self.page.keyboard.type(expression)

        save_btn = self.page.locator("button").filter(has_text="Save").first
        if save_btn.count() == 0:
            save_btn = self.page.locator("button").filter(has_text="Create").first
        save_btn.click()

        try:
            name_input.wait_for(state="hidden", timeout=1000)
            return None
        except Exception:
            modal = self.page.locator(
                "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
            )
            error_container = modal.locator(".pluto--status-error").first.locator("..")
            if error_container.count() > 0:
                error_text = error_container.inner_text().strip()
                self.close_modal()
                return error_text
            self.close_modal()
            return "Unknown error"
        finally:
            self.hide_channels()

    def edit_calculated(self, name: ChannelName, new_expression: str) -> None:
        """Edit a calculated channel's expression via context menu.

        :param name: The name of the calculated channel to edit.
        :param new_expression: The new calculation expression.
        """
        self._right_click_channel(name)

        edit_option = self.page.get_by_text("Edit calculation", exact=True).first
        edit_option.wait_for(state="visible", timeout=5000)
        edit_option.click(timeout=1000)

        editor = self.page.locator(".monaco-editor")
        editor.wait_for(state="visible", timeout=5000)
        editor.click()
        self.page.locator(".monaco-editor.focused").wait_for(
            state="visible", timeout=2000
        )
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.wait_for_timeout(50)
        self.page.keyboard.type(new_expression)

        self.page.locator("button").filter(has_text="Save").first.click()
        editor.wait_for(state="hidden", timeout=5000)

        self.hide_channels()

    def set_alias(self, name: ChannelName, alias: str) -> None:
        """Set an alias for a channel under the active range via context menu.

        Note: Requires an active range to be set before calling this method.

        :param name: The name of the channel to set an alias for.
        :param alias: The alias to set for the channel.
        """
        self._right_click_channel(name)

        set_alias_option = self.page.get_by_text(re.compile(r"Set alias under")).first
        set_alias_option.wait_for(state="visible", timeout=5000)
        set_alias_option.click(timeout=1000)

        self.page.keyboard.type(alias)
        self.console.ENTER
        self.page.locator("input.pluto-text--editable").wait_for(
            state="hidden", timeout=5000
        )

        self.hide_channels()

    def clear_alias(self, name: ChannelName) -> None:
        """Clear an alias for a channel under the active range via context menu.

        Note: Requires an active range to be set before calling this method.

        :param name: The name of the channel to clear the alias for.
        """
        self._right_click_channel(name)

        clear_alias_option = self.page.get_by_text(
            re.compile(r"Remove alias under")
        ).first
        clear_alias_option.wait_for(state="visible", timeout=5000)
        clear_alias_option.click(timeout=1000)
        clear_alias_option.wait_for(state="hidden", timeout=5000)

        self.hide_channels()

    def hard_reload(self) -> None:
        """Trigger hard reload from channel context menu.

        Right-clicks any visible channel and selects "Reload Console" option.
        This will reload the entire console.
        """
        self.show_channels()
        item = self.channels_list.first
        item.click(button="right")

        reload_option = self.page.get_by_text("Reload Console", exact=True).first
        reload_option.wait_for(state="visible", timeout=2000)
        reload_option.click(timeout=1000)

        self.page.wait_for_load_state("load", timeout=30000)
        self.page.wait_for_load_state("networkidle", timeout=30000)

    def open_plot(self, name: ChannelName) -> None:
        """Open a channel's plot by double-clicking it in the sidebar.

        :param name: The name of the channel to open.
        """
        self.show_channels()
        item = self._find_channel_item(name)
        if item is None:
            raise ValueError(f"Channel {name} not found")
        item.dblclick()
        self.page.wait_for_timeout(500)
        self.hide_channels()

    def group(self, names: ChannelNames, group_name: str) -> None:
        """Group multiple channels together via context menu.

        :param names: List of channel names to group.
        :param group_name: The name for the new group.
        """
        import platform

        if len(names) < 2:
            raise ValueError("At least 2 channels are required to create a group")

        self.show_channels()

        # Find and select all channels
        modifier = "Meta" if platform.system() == "Darwin" else "Control"
        first_item = True

        for name in names:
            for item in self.channels_list.all():
                if item.is_visible():
                    channel_name_element = item.locator("p.pluto-text--editable")
                    text = channel_name_element.inner_text().strip()
                    if text == name:
                        if first_item:
                            item.click()
                            first_item = False
                        else:
                            # Cmd/Ctrl+click to add to selection
                            self.page.keyboard.down(modifier)
                            item.click()
                            self.page.keyboard.up(modifier)
                        break

        self.page.wait_for_timeout(200)

        # Right-click to open context menu on the last selected item
        for item in self.channels_list.all():
            if item.is_visible():
                channel_name_element = item.locator("p.pluto-text--editable")
                text = channel_name_element.inner_text().strip()
                if text == names[-1]:
                    item.click(button="right")
                    break

        self.page.wait_for_timeout(200)

        # Click "Group" option - this creates a folder in the toolbar
        self.page.get_by_text("Group Selection", exact=True).first.click()
        self.page.wait_for_timeout(500)

        # A folder is created with an inline editable name - look for the input
        # The folder/group should have an editable text input active
        editable_input = self.page.locator("input.pluto-text__input--editable").first
        if editable_input.count() > 0 and editable_input.is_visible():
            editable_input.fill(group_name)
            self.page.keyboard.press("Enter")
        else:
            # If no input is visible, try to find and click the new folder to rename it
            self.page.keyboard.type(group_name)
            self.page.keyboard.press("Enter")

        self.page.wait_for_timeout(300)
        self.hide_channels()

    def copy_link(self, name: ChannelName) -> str:
        """Copy link to a channel via context menu.

        :param name: The name of the channel to copy link for.
        :returns: The copied link (if clipboard access is available).
        """
        self._right_click_channel(name)
        self.page.wait_for_timeout(500)

        self.page.get_by_text("Copy Link").first.click()
        self.page.wait_for_timeout(200)

        self.hide_channels()

        try:
            link: str = str(self.page.evaluate("navigator.clipboard.readText()"))
            return link
        except Exception:
            return ""

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

        item = self._right_click_channel(old_name)

        rename_option = self.page.get_by_text("Rename", exact=True).first
        self.page.wait_for_timeout(200)
        rename_option.click(timeout=1000)

        channel_name_element = item.locator("p.pluto-text--editable")
        channel_name_element.click()
        channel_name_element.fill(new_name)
        self.page.keyboard.press("Enter")
        self.page.wait_for_timeout(100)

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

        self._right_click_channel(name)

        delete_option = self.page.get_by_text("Delete", exact=True).first
        delete_option.wait_for(state="visible", timeout=5000)
        delete_option.click()

        modal = self.page.locator(
            "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
        )
        try:
            modal.wait_for(state="visible", timeout=2000)
            modal_delete_btn = modal.get_by_role("button", name="Delete", exact=True)
            modal_delete_btn.click()
            modal.wait_for(state="hidden", timeout=5000)
        except Exception:
            pass

        for i, notification in enumerate(self.console.notifications.check()):
            message = notification.get("message", "")
            description = notification.get("description", "")
            if message == "Failed to delete Channel" and name in description:
                self.console.notifications.close(i)
                raise RuntimeError(f"{message} {name}, {description}")

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

    def open_plot_by_search(self, name: ChannelName) -> None:
        """Open a channel plot by searching its name in the command palette.

        :param name: The name of the channel to search for and open.
        """
        self.console.search_palette(name)

        line_plot = self.page.locator(".pluto-line-plot")
        line_plot.first.wait_for(state="visible", timeout=5000)

    def close_modal(self) -> None:
        """Close any open modal by clicking the close button."""
        close_button = self.page.locator(
            ".pluto-dialog__dialog button:has(svg.pluto-icon--close)"
        ).first
        close_button.click(timeout=2000)

        modal = self.page.locator(
            "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
        )
        modal.wait_for(state="hidden", timeout=2000)

    def open_create_modal(self) -> None:
        """Open the Create Channel modal via command palette.

        The modal will be visible after this method returns.
        Use close_modal() to close it.
        """
        self.console.command_palette("Create a Channel")

        modal = self.page.locator(
            "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
        )
        modal.wait_for(state="visible", timeout=5000)

        name_input = self.page.locator("input[placeholder='Name']")
        name_input.wait_for(state="visible", timeout=2000)

    def open_create_calculated_modal(self) -> None:
        """Open the Create Calculated Channel modal via command palette.

        The modal will be visible after this method returns.
        Use close_modal() to close it.
        """
        self.console.command_palette("Create a Calculated Channel")

        modal = self.page.locator(
            "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
        )
        modal.wait_for(state="visible", timeout=5000)

        name_input = self.page.locator("input[placeholder='Name']")
        name_input.wait_for(state="visible", timeout=2000)

        editor = self.page.locator(".monaco-editor")
        editor.wait_for(state="visible", timeout=2000)
