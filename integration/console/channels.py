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
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError
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

from console.context_menu import ContextMenu
from console.layout import LayoutClient
from console.notifications import NotificationsClient
from console.tree import Tree


class ChannelClient:
    """Console channel client for managing channels via the UI.

    Provides methods for creating, renaming, deleting, and organizing channels
    through the Console sidebar and command palette.
    """

    ITEM_PREFIX = "channel:"
    ICON_NAME = "channel"

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
    ):
        self.layout = layout
        self.ctx_menu = ContextMenu(layout.page)
        self.notifications = NotificationsClient(layout.page)
        self.tree = Tree(layout.page)
        self.client = client

    def _get_channels_button(self) -> Locator:
        """Get the channels button in the sidebar."""
        return self.layout.page.locator("button.console-main-nav__item").filter(
            has=self.layout.page.locator(f"svg.pluto-icon--{self.ICON_NAME}")
        )

    def _get_channels_pane(self) -> Locator:
        """Get the channels pane header."""
        return self.layout.page.locator("text=Channels").first

    def show_channels(self) -> None:
        """Show the channels pane in the sidebar if not already visible."""
        channels_pane = self._get_channels_pane()
        if channels_pane.is_visible():
            return
        self._get_channels_button().click(force=True, timeout=5000)
        channels_pane.wait_for(state="visible", timeout=5000)
        self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']").first.wait_for(
            state="visible", timeout=5000
        )

    def hide_channels(self) -> None:
        """Hide the channels pane in the sidebar if currently visible."""
        if self._get_channels_pane().is_visible():
            self._get_channels_button().click(force=True, timeout=2000)

    def _find_channel_item(
        self, name: ChannelName, retry_with_refresh: bool = True
    ) -> Locator | None:
        """Find a channel item in the list by name.

        :param name: The channel name to find.
        :param retry_with_refresh: If True and channel not found, refresh the
            channels pane and try again (handles cases where list is stale).
        :returns: The channel item Locator, or None if not found.
        """
        item = self.tree.find_by_name(self.ITEM_PREFIX, str(name))
        if item is None and retry_with_refresh:
            self.hide_channels()
            self.layout.page.wait_for_timeout(100)
            self.show_channels()
            return self.tree.find_by_name(self.ITEM_PREFIX, str(name))
        return item

    def _right_click_channel(self, name: ChannelName) -> Locator:
        """Find a channel and right-click it to open context menu.

        :param name: The channel name to right-click.
        :returns: The channel item Locator that was right-clicked.
        :raises ValueError: If channel not found in the channel list.
        """
        self.show_channels()
        item = self._find_channel_item(name)
        if item is None:
            raise ValueError(f"Channel {name} not found")
        self.ctx_menu.open_on(item)
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

        self.open_create_modal()
        self.layout.fill_input_field("Name", name)
        # Set virtual if needed
        if virtual:
            self.layout.click_checkbox("Virtual")
        # Configure as index or regular channel
        if is_index:
            self.layout.click_checkbox("Is Index")
        else:
            if index == 0:
                raise ValueError("Index must be provided if is_index is False")

            # Set data type
            data_type_str = str(DataType(data_type))

            self.layout.click_btn("Data Type")
            self.layout.select_from_dropdown(data_type_str, "Search Data Types")

            # Set index - index should be the channel name
            self.layout.click_btn("Index")
            self.layout.select_from_dropdown(index, "Search Channels")

        self.layout.page.get_by_role("button", name="Create", exact=True).click()
        modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
        modal.wait_for(state="hidden", timeout=5000)
        self.show_channels()
        for _ in range(20):
            if self._find_channel_item(name, retry_with_refresh=False) is not None:
                break
            self.layout.page.wait_for_timeout(50)
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

            # Open command palette for first channel
            if i == 0:
                self.layout.command_palette("Create a Channel")
                # Wait for modal to appear
                self.layout.page.wait_for_selector(
                    self.layout.MODAL_SELECTOR,
                    timeout=5000,
                )
            else:
                modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
                modal_count = modal.count()
                if modal_count == 0:
                    raise RuntimeError(
                        "Modal closed between channel creations despite 'Create More'"
                    )

            # Fill channel name (use same pattern as create method)
            name_input = self.layout.page.get_by_role("textbox", name="Name")
            name_input.wait_for(state="visible", timeout=5000)
            name_input.fill(name)

            # Set virtual if needed
            if virtual:
                self.layout.click_checkbox("Virtual")

            # Configure as index or regular channel
            if is_index:
                self.layout.click_checkbox("Is Index")
            else:
                if index == 0:
                    raise ValueError(
                        f"Index must be provided for non-index channel: {name}"
                    )

                # Set data type
                data_type_str = str(DataType(data_type))
                self.layout.click_btn("Data Type")
                self.layout.select_from_dropdown(data_type_str, "Search Data Types")

                # Set index
                self.layout.click_btn("Index")
                self.layout.select_from_dropdown(index_str, "Search Channels")

            # Check "Create More" for all but the last channel
            is_last = i == len(channels) - 1
            create_more_checkbox = (
                self.layout.page.locator("text=Create More")
                .locator("..")
                .locator("input[type='checkbox']")
                .first
            )

            if not is_last:
                # Ensure "Create More" is checked
                is_checked = create_more_checkbox.is_checked()
                if not is_checked:
                    create_more_checkbox.click()
            else:
                # Ensure "Create More" is unchecked for last channel
                is_checked = create_more_checkbox.is_checked()
                if is_checked:
                    create_more_checkbox.click()

            self.layout.page.get_by_role("button", name="Create", exact=True).click()
            created_channels.append(name)

            if not is_last:
                modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
                modal_count = modal.count()
                if modal_count == 0:
                    raise RuntimeError(
                        "Modal closed after creating channel with 'Create More' checked"
                    )

                name_input_after = self.layout.page.get_by_role("textbox", name="Name")

                name_input_after.wait_for(state="visible", timeout=5000)
                # Wait for input to be cleared (form reset)
                for attempt in range(30):
                    try:
                        current_val = name_input_after.input_value()
                        if current_val == "":
                            break
                    except Exception:
                        pass
                    self.layout.page.wait_for_timeout(100)
                else:
                    raise RuntimeError(
                        "Form did not reset after creating channel with 'Create More'"
                    )

        self.hide_channels()
        return created_channels

    def create_calculated(self, *, name: ChannelName, expression: str) -> str | None:
        """Creates a calculated channel via console UI.

        :param name: The name for the calculated channel.
        :param expression: The calculation expression (e.g., "channel_a * 2").
        :returns: None if successful, error message string if failed.
        """
        self.open_create_calculated_modal()

        name_input = self.layout.page.locator("input[placeholder='Name']")
        name_input.fill(name)

        editor = self.layout.page.locator(".monaco-editor")
        editor.click()
        self.layout.page.locator(".monaco-editor.focused").wait_for(
            state="visible", timeout=2000
        )
        sy.sleep(0.2)
        self.layout.page.keyboard.type(expression)

        save_btn = self.layout.page.locator("button").filter(has_text="Save").first
        if save_btn.count() == 0:
            save_btn = (
                self.layout.page.locator("button").filter(has_text="Create").first
            )
        save_btn.click()

        try:
            name_input.wait_for(state="hidden", timeout=3000)
            return None
        except PlaywrightTimeoutError:
            modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
            modal_count = modal.count()

            if modal_count > 0:
                modal_text = modal.inner_text()
                if "Failed to update calculated channel" in modal_text:
                    error_start = modal_text.find("Failed to update calculated channel")
                    error_section = modal_text[error_start:]
                    for delimiter in ["\n\nCreate More", "\n\nSave"]:
                        if delimiter in error_section:
                            error_section = error_section[
                                : error_section.find(delimiter)
                            ]
                            break
                    error_text = error_section.strip()
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
        self.ctx_menu.click_option("Edit calculation")

        editor = self.layout.page.locator(".monaco-editor")
        editor.wait_for(state="visible", timeout=5000)
        editor.click()
        self.layout.page.locator(".monaco-editor.focused").wait_for(
            state="visible", timeout=2000
        )
        self.layout.page.wait_for_timeout(50)
        self.layout.select_all_and_type(new_expression)

        self.layout.page.locator("button").filter(has_text="Save").first.click()
        editor.wait_for(state="hidden", timeout=5000)

        self.hide_channels()

    def set_alias(self, *, name: ChannelName, alias: str) -> None:
        """Set an alias for a channel under the active range via context menu.

        Note: Requires an active range to be set before calling this method.

        :param name: The name of the channel to set an alias for.
        :param alias: The alias to set for the channel.
        """
        self._right_click_channel(name)
        self.ctx_menu.click_option("Set alias under", exact=False)

        self.layout.page.keyboard.type(alias)
        self.layout.press_enter()
        self.layout.page.locator("input.pluto-text--editable").wait_for(
            state="hidden", timeout=5000
        )

        self.hide_channels()

    def clear_alias(self, name: ChannelName) -> None:
        """Clear an alias for a channel under the active range via context menu.

        Note: Requires an active range to be set before calling this method.

        :param name: The name of the channel to clear the alias for.
        """
        self._right_click_channel(name)
        self.ctx_menu.click_option("Remove alias under", exact=False)

        self.hide_channels()

    def hard_reload(self) -> None:
        """Trigger hard reload from channel context menu.

        Right-clicks any visible channel and selects "Reload Console" option.
        This will reload the entire console.
        """
        self.show_channels()
        items = self.tree.find_by_prefix(self.ITEM_PREFIX)
        if not items:
            raise ValueError("No channels found to trigger reload")
        self.ctx_menu.action(items[0], "Reload Console")

        self.layout.page.wait_for_load_state("load", timeout=30000)
        self.layout.page.wait_for_load_state("networkidle", timeout=30000)

    def group(self, *, names: ChannelNames, group_name: str) -> None:
        """Group multiple channels together via context menu.

        :param names: List of channel names to group.
        :param group_name: The name for the new group.
        """
        if len(names) < 2:
            raise ValueError("At least 2 channels are required to create a group")

        self.show_channels()

        # Select all channels (first one normal click, rest with Ctrl+Click)
        last_item = None
        for i, name in enumerate(names):
            item = self.tree.find_by_name(self.ITEM_PREFIX, name)
            if item is None:
                raise ValueError(f"Channel {name} not found")
            if i == 0:
                item.click()
            else:
                item.click(modifiers=["ControlOrMeta"])
            last_item = item

        # Right-click last item and select "Group Selection"
        assert last_item is not None
        self.ctx_menu.action(last_item, "Group Selection")

        editable_input = self.layout.page.locator(
            "input.pluto-text__input--editable"
        ).first
        try:
            editable_input.wait_for(state="visible", timeout=500)
            editable_input.fill(group_name)
            self.layout.press_enter()
        except Exception:
            self.layout.page.keyboard.type(group_name)
            self.layout.press_enter()

        self.hide_channels()

    def copy_link(self, name: ChannelName) -> str:
        """Copy link to a channel via context menu.

        :param name: The name of the channel to copy link for.
        :returns: The copied link (if clipboard access is available).
        """
        self._right_click_channel(name)
        self.ctx_menu.click_option("Copy link")
        self.hide_channels()
        return self.layout.read_clipboard()

    def exists(self, name: ChannelName) -> bool:
        """
        Checks if a channel with the given name exists.

        :param name: The name of the channel to check.
        :returns: True if the channel exists, False otherwise.
        """
        self.show_channels()
        channel_name_str = str(name)
        selector = f"div[id^='{self.ITEM_PREFIX}'] p.pluto-text--editable:has-text('{channel_name_str}')"
        try:
            self.layout.page.wait_for_selector(selector, state="visible", timeout=500)
            return True
        except PlaywrightTimeoutError:
            return False
        finally:
            self.hide_channels()

    def wait_for_channel_removed(self, name: ChannelName) -> None:
        """Wait for a channel to be removed from the channel list."""
        self.show_channels()
        self.tree.wait_for_removal(self.ITEM_PREFIX, str(name), exact=True)
        self.hide_channels()

    def wait_for_channels(
        self, names: ChannelNames, timeout: sy.CrudeTimeSpan = 10.0
    ) -> bool:
        """Wait for one or more channels to appear in the console UI.

        Uses Playwright's wait_for_selector to efficiently wait for specific channels.

        :param names: The name(s) of the channel(s) to wait for.
        :param timeout: Maximum time to wait in seconds (default: 10.0).
        :returns: True if all channels exist, False if timeout reached.
        """
        normalized_names = normalize_channel_params(names)
        timeout_ms = int(timeout * 1000)

        self.show_channels()

        try:
            for name in normalized_names.channels:
                channel_name_str = str(name)
                selector = f"div[id^='{self.ITEM_PREFIX}'] p.pluto-text--editable:has-text('{channel_name_str}')"
                try:
                    self.layout.page.wait_for_selector(
                        selector, state="visible", timeout=timeout_ms
                    )
                except PlaywrightTimeoutError:
                    self.hide_channels()
                    return False

            self.hide_channels()
            return True
        except PlaywrightTimeoutError:
            self.hide_channels()
            return False

    def rename(self, *, names: ChannelNames, new_names: ChannelNames) -> bool:
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
                self._rename_single_channel(
                    old_name=str(old_name), new_name=str(new_name)
                )

            return True
        except Exception:
            return False

    def _rename_single_channel(self, *, old_name: str, new_name: str) -> None:
        """Rename a single channel via the context menu.

        Args:
            old_name: The current name of the channel.
            new_name: The new name for the channel.
        """
        item = self._right_click_channel(old_name)
        self.ctx_menu.click_option("Rename")

        channel_name_element = item.locator("p.pluto-text--editable")
        channel_name_element.click()
        channel_name_element.fill(new_name)
        self.layout.press_enter()

        self.hide_channels()

    def delete(self, names: ChannelNames) -> None:
        """Deletes one or more channels via console UI."""
        normalized_names = normalize_channel_params(names)
        for name in normalized_names.channels:
            self._delete_single_channel(str(name))

    def _delete_single_channel(self, name: str) -> None:
        """Delete a single channel via the context menu."""
        self.show_channels()
        item = self._find_channel_item(name)
        if item is None:
            raise ValueError(f"Channel {name} not found")

        self.layout.delete_with_confirmation(item)

        for i, notification in enumerate(self.notifications.check()):
            message = notification.get("message", "")
            description = notification.get("description", "")
            if message == "Failed to delete Channel" and name in description:
                self.notifications.close(i)
                raise RuntimeError(f"{message} {name}, {description}")

        self.tree.wait_for_removal(self.ITEM_PREFIX, name, exact=True)
        self.hide_channels()

    def list_all(self) -> list[ChannelName]:
        """List all visible channels in the sidebar.

        Returns:
            A list of channel names currently visible in the channels pane.
        """
        self.show_channels()
        return self.tree.list_names(self.ITEM_PREFIX)

    def close_modal(self) -> None:
        """Close any open modal by clicking the close button."""
        close_button = self.layout.page.locator(
            ".pluto-dialog__dialog button:has(svg.pluto-icon--close)"
        ).first
        close_button.click(timeout=2000)

        modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
        modal.wait_for(state="hidden", timeout=2000)

    def open_create_modal(self) -> None:
        """Open the Create Channel modal via command palette.

        The modal will be visible after this method returns.
        Use close_modal() to close it.
        """
        self.layout.command_palette("Create a Channel")

        modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)

        name_input = self.layout.page.locator("input[placeholder='Name']")
        name_input.wait_for(state="visible", timeout=2000)

    def open_create_calculated_modal(self) -> None:
        """Open the Create Calculated Channel modal via command palette.

        The modal will be visible after this method returns.
        Use close_modal() to close it.
        """
        self.layout.command_palette("Create a Calculated Channel")

        modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)

        name_input = self.layout.page.locator("input[placeholder='Name']")
        name_input.wait_for(state="visible", timeout=2000)

        editor = self.layout.page.locator(".monaco-editor")
        editor.wait_for(state="visible", timeout=2000)
