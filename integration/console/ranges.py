#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError
from playwright.sync_api import expect

from console.context_menu import ContextMenu
from console.layout import LayoutClient
from console.notifications import NotificationsClient
from console.tree import Tree
from framework.utils import get_results_path, rgb_to_hex


class RangesClient:
    """Console ranges client for managing ranges via the UI.

    The ranges toolbar shows only favorited ranges.
    The Range Explorer shows all persisted ranges.
    """

    TOOLBAR_ITEM_SELECTOR = ".console-range-list-item"
    EXPLORER_ITEM_SELECTOR = ".console-range__list-item"
    CREATE_MODAL_SELECTOR = ".console-range-create-layout"
    NAME_INPUT_PLACEHOLDER = "Range Name"

    def __init__(
        self,
        layout: LayoutClient,
    ):
        self.layout = layout
        self.ctx_menu = ContextMenu(layout.page)
        self.notifications = NotificationsClient(layout.page)
        self.tree = Tree(layout.page)

    # ── Private Helpers ──────────────────────────────────────────────────

    def _toolbar_ctx_menu_action(self, name: str, action_text: str) -> None:
        """Show the toolbar, find an item by name, and perform a context menu action."""
        self.show_toolbar()
        item = self.get_toolbar_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, action_text)

    def _set_time_in_overview(
        self,
        index: int,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Set a time in the range overview by button index (0=start, 1=end)."""
        time_range = self.layout.page.locator(".console-time-range")
        btn = time_range.locator("button").nth(index)
        btn.wait_for(state="visible", timeout=5000)
        self._fill_datetime_picker(btn, year, month, day, hour, minute, second)

    def _open_labels_dropdown(self) -> Locator:
        """Open the labels dropdown in the range overview and return the dialog."""
        labels_row = self.layout.page.get_by_text("Labels", exact=True).locator("..")
        add_button = labels_row.locator("button").last
        add_button.wait_for(state="visible", timeout=2000)
        add_button.click()
        dropdown = self.layout.page.locator(".pluto-dialog__dialog.pluto--visible")
        dropdown.wait_for(state="visible", timeout=5000)
        return dropdown

    def _pick_stage_from_dropdown(self, stage_button: Locator, stage: str) -> None:
        """Click a stage button and select a stage from the dropdown."""
        stage_button.click()
        dropdown = self.layout.page.locator(".pluto-list__item").filter(has_text=stage)
        dropdown.click(timeout=2000)
        dropdown.wait_for(state="hidden", timeout=2000)

    # ── Public API ───────────────────────────────────────────────────────

    def open_from_search(self, name: str) -> None:
        """Open a range overview by searching its name in the command palette.

        Args:
            name: Name of the range to search for and open.
        """
        self.layout.search_palette(name)
        name_input = self.layout.page.locator("input[placeholder='Name']").first
        name_input.wait_for(state="visible", timeout=5000)
        expect(name_input).to_have_value(name, timeout=5000)

    def show_toolbar(self) -> None:
        """Show the ranges toolbar in the left sidebar (favorites only)."""
        self.layout.show_resource_toolbar("range")

    def hide_toolbar(self) -> None:
        """Hide the ranges toolbar."""
        self.layout.close_left_toolbar()

    def open_explorer(self) -> None:
        """Open the Range Explorer page (shows all ranges)."""
        self.layout.command_palette("Open the Range Explorer")
        self.layout.page.get_by_text("All Ranges").wait_for(
            state="visible", timeout=5000
        )

    def get_toolbar_item(self, name: str) -> Locator:
        """Get a range item locator from the toolbar by name."""
        return self.layout.get_list_item(self.TOOLBAR_ITEM_SELECTOR, name)

    def get_explorer_item(self, name: str) -> Locator:
        """Get a range item locator from the explorer by name."""
        return self.layout.get_list_item(self.EXPLORER_ITEM_SELECTOR, name)

    def get_toolbar_item_time(self, name: str) -> str:
        """Get the displayed time text from a toolbar range item.

        Args:
            name: The name of the range.

        Returns:
            The time range text (e.g. "Today 16:23:35 → 16:23:35").
        """
        self.show_toolbar()
        item = self.get_toolbar_item(name)
        return item.locator("small.pluto-text--small").first.inner_text()

    def get_explorer_item_time(self, name: str) -> str:
        """Get the displayed time text from an explorer range item.

        Args:
            name: The name of the range.

        Returns:
            The time range text (e.g. "Jan 1 00:00:00 → Jan 2 00:00:00").
        """
        item = self.get_explorer_item(name)
        return item.locator("small.pluto-text--small").first.inner_text()

    def exists_in_toolbar(self, name: str) -> bool:
        """Check if a range exists in the toolbar (is favorited)."""
        self.show_toolbar()
        return self.layout.locator_exists(self.get_toolbar_item(name))

    def exists_in_explorer(self, name: str) -> bool:
        """Check if a range exists in the explorer."""
        return self.layout.locator_exists(self.get_explorer_item(name))

    def wait_for_removed_from_toolbar(self, name: str) -> None:
        """Wait for a range to be removed from the toolbar."""
        self.show_toolbar()
        self.layout.wait_for_hidden(self.get_toolbar_item(name))

    def wait_for_removed_from_explorer(self, name: str) -> None:
        """Wait for a range to be removed from the explorer."""
        self.layout.wait_for_hidden(self.get_explorer_item(name))

    def create(
        self,
        name: str,
        *,
        persisted: bool = True,
        parent: str | None = None,
        labels: list[str] | None = None,
        stage: str | None = None,
    ) -> None:
        """Create a new range via the command palette.

        Args:
            name: The name for the new range.
            persisted: If True, saves to Synnax server. If False, saves locally only.
            parent: Optional parent range name to set.
            labels: Optional list of label names to add.
            stage: Optional stage to set ("To Do", "In Progress", "Completed").
        """
        self.layout.command_palette("Create a range")
        modal = self.layout.page.locator(self.CREATE_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        self._fill_create_modal(
            name, persisted=persisted, parent=parent, labels=labels, stage=stage
        )

    def _fill_create_modal(
        self,
        name: str,
        *,
        persisted: bool = True,
        parent: str | None = None,
        labels: list[str] | None = None,
        stage: str | None = None,
    ) -> None:
        """Fill and submit the range creation modal.

        Assumes the modal is already open and visible.
        """
        modal = self.layout.page.locator(self.CREATE_MODAL_SELECTOR)
        name_input = self.layout.page.locator(
            f"input[placeholder='{self.NAME_INPUT_PLACEHOLDER}']"
        )
        name_input.fill(name)

        if stage is not None:
            stage_button = (
                modal.locator("button")
                .filter(has_text="To Do")
                .or_(modal.locator("button").filter(has_text="In Progress"))
                .or_(modal.locator("button").filter(has_text="Completed"))
                .first
            )
            self._pick_stage_from_dropdown(stage_button, stage)

        if parent is not None:
            parent_button = modal.locator("button").filter(has_text="Select a range")
            parent_button.click()
            search_input = self.layout.page.locator(
                "input[placeholder='Search ranges...']"
            )
            search_input.fill(parent)
            self.layout.page.locator(".pluto-range__list-item").filter(
                has_text=parent
            ).click(timeout=5000)

        if labels is not None:
            label_button = self.layout.page.get_by_text("Select labels", exact=True)
            label_button.click(timeout=5000)
            for label_name in labels:
                label_item = (
                    self.layout.page.locator(".pluto-list__item")
                    .filter(has_text=label_name)
                    .first
                )
                try:
                    label_item.wait_for(state="visible", timeout=3000)
                    label_item.click(timeout=2000)
                except PlaywrightTimeoutError:
                    all_labels = self.layout.page.locator(".pluto-list__item").all()
                    available_labels = [
                        lbl.text_content() for lbl in all_labels if lbl.is_visible()
                    ]
                    raise RuntimeError(
                        f"Error selecting label '{label_name}'. Available labels: {available_labels}."
                    )
            self.layout.press_escape()

        if persisted:
            save_button = self.layout.page.get_by_role("button", name="Save to Synnax")
        else:
            save_button = self.layout.page.get_by_role("button", name="Save Locally")

        save_button.click(timeout=2000)
        modal.wait_for(state="hidden", timeout=5000)

    def set_active(self, name: str) -> None:
        """Set a range as the active range (from toolbar)."""
        self.show_toolbar()
        item = self.get_toolbar_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()

    def rename_from_explorer(self, old_name: str, new_name: str) -> None:
        """Rename a range via modal dialog from the explorer."""
        self.layout.rename_with_modal(self.get_explorer_item(old_name), new_name)

    def delete_from_explorer(self, name: str) -> None:
        """Delete a range via context menu in the explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.layout.delete_with_confirmation(item)
        item.wait_for(state="hidden", timeout=5000)

    def favorite_from_explorer(self, name: str) -> None:
        """Add a range to favorites via context menu in the explorer."""
        self.layout.hide_visualization_toolbar()
        self.notifications.close_all()
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.open_on(item)
        add_btn = self.layout.page.get_by_text("Add to favorites", exact=True)
        remove_btn = self.layout.page.get_by_text("Remove from favorites", exact=True)
        add_btn.or_(remove_btn).wait_for(state="visible", timeout=2000)
        if remove_btn.is_visible():
            self.ctx_menu.close()
            return
        self.ctx_menu.click_option("Add to favorites")

    def unfavorite_from_toolbar(self, name: str) -> None:
        """Remove a range from favorites via context menu in the toolbar."""
        self._toolbar_ctx_menu_action(name, "Remove from favorites")
        self.wait_for_removed_from_toolbar(name)

    def save_to_synnax_from_toolbar(self, name: str) -> None:
        """Save a local range to Synnax via context menu in the toolbar."""
        self._toolbar_ctx_menu_action(name, "Save to Synnax")

    def add_to_new_plot_from_toolbar(self, name: str) -> None:
        """Add a range to a new line plot via context menu in the toolbar."""
        self._toolbar_ctx_menu_action(name, "Add to new plot")

    def add_to_active_plot_from_toolbar(self, name: str) -> None:
        """Add a range to the active line plot via context menu in the toolbar."""
        self._toolbar_ctx_menu_action(name, "Add to active plot")

    def favorite(self, name: str) -> None:
        """Favorite a range by opening its overview and clicking the favorite button.

        Args:
            name: The name of the range to favorite.
        """
        self.open_from_search(name)

        favorite_btn = self.layout.page.locator("button.console-favorite-button")
        favorite_btn.wait_for(state="visible", timeout=5000)

        button_class = favorite_btn.get_attribute("class") or ""
        is_favorited = "console--favorite" in button_class

        if not is_favorited:
            favorite_btn.click(force=True)
            self.layout.page.locator(
                "button.console-favorite-button.console--favorite"
            ).wait_for(state="visible", timeout=2000)

        self.layout.close_tab(name)

    def open_overview_from_explorer(self, name: str) -> None:
        """Open the range overview/details page from explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.dblclick()

    def navigate_to_parent(self, parent_name: str) -> None:
        """Navigate to parent range from current range overview.

        Args:
            parent_name: The name of the parent range to navigate to.
        """
        parent_button = self.layout.page.get_by_role("button").filter(
            has_text=parent_name
        )
        parent_button.click(timeout=5000)

    def wait_for_overview(self, name: str) -> None:
        """Wait for the range overview to show a specific range."""
        name_input = self.layout.page.locator("input[placeholder='Name']").first
        name_input.wait_for(state="visible", timeout=5000)
        expect(name_input).to_have_value(name, timeout=5000)

    def is_overview_showing(self, name: str) -> bool:
        """Check if the range overview is showing a specific range.

        Args:
            name: The name of the range to check for.

        Returns:
            True if the overview shows the range name in the header.
        """
        header = self.layout.page.locator("input[placeholder='Name']").first
        if not header.is_visible():
            return False
        return header.input_value() == name

    def _navigate_calendar_to_year(self, calendar: Locator, target_year: int) -> None:
        """Navigate the calendar picker to the target year."""
        year_row = calendar.locator("> .pluto-flex").nth(1)
        while True:
            current_year = int(year_row.locator("small").inner_text())
            if current_year == target_year:
                break
            if current_year > target_year:
                year_row.locator("button").first.click()
            else:
                year_row.locator("button").last.click()

    def _navigate_calendar_to_month(self, calendar: Locator, target_month: str) -> None:
        """Navigate the calendar picker to the target month."""
        month_row = calendar.locator(".pluto-calendar-header")
        while True:
            current_month = month_row.locator(
                ".pluto-calendar-header__month"
            ).inner_text()
            if current_month == target_month:
                break
            month_row.locator("button").first.click()

    def _select_time_value(self, time_list: Locator, value: int) -> None:
        """Select a value from a time list by clicking the item with matching id."""
        item = time_list.locator(f".pluto-list__item[id='{value}']")
        item.scroll_into_view_if_needed()
        item.click()

    def _fill_datetime_picker(
        self,
        field: Locator,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Fill a datetime input using the datetime picker modal."""
        field.click()
        modal = self.layout.page.locator(".pluto-datetime-modal")
        modal.wait_for(state="visible", timeout=5000)

        picker = modal.locator(".pluto-datetime-picker")
        calendar = picker.locator(".pluto-calendar")

        self._navigate_calendar_to_year(calendar, year)
        self._navigate_calendar_to_month(calendar, month)

        day_btn = calendar.get_by_role("button", name=str(day), exact=True)
        day_btn.click()

        time_lists = picker.locator(".pluto-time-list")
        self._select_time_value(time_lists.nth(0), hour)
        self._select_time_value(time_lists.nth(1), minute)
        self._select_time_value(time_lists.nth(2), second)

        done_btn = self.layout.page.get_by_role("button", name="Done")
        done_btn.click()
        modal.wait_for(state="hidden", timeout=5000)

    def set_start_time_in_overview(
        self,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Set the start time in the range overview."""
        self._set_time_in_overview(0, year, month, day, hour, minute, second)

    def set_end_time_in_overview(
        self,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Set the end time in the range overview."""
        self._set_time_in_overview(1, year, month, day, hour, minute, second)

    def set_stage_in_overview(self, stage: str) -> None:
        """Set the stage in the range overview.

        Args:
            stage: The stage to set ("To Do", "In Progress", "Completed").
        """
        stage_button = (
            self.layout.page.locator("button")
            .filter(has_text="To Do")
            .or_(self.layout.page.locator("button").filter(has_text="In Progress"))
            .or_(self.layout.page.locator("button").filter(has_text="Completed"))
            .first
        )
        self._pick_stage_from_dropdown(stage_button, stage)

    def add_label_in_overview(self, label_name: str) -> None:
        """Add a label to the range in the overview.

        Args:
            label_name: The name of the label to add.
        """
        dropdown = self._open_labels_dropdown()
        item = dropdown.locator(".pluto-list__item").filter(has_text=label_name).first
        try:
            item.wait_for(state="visible", timeout=5000)
            item.click(timeout=2000)
        except PlaywrightTimeoutError as e:
            all_items = dropdown.locator(".pluto-list__item").all()
            available_labels = [
                lbl.text_content() for lbl in all_items if lbl.is_visible()
            ]
            raise PlaywrightTimeoutError(
                f"Label '{label_name}' not found in dropdown. Available: {available_labels}"
            ) from e
        self.layout.press_escape()
        dropdown.wait_for(state="hidden", timeout=5000)

    def remove_label_in_overview(self, label_name: str) -> None:
        """Remove a label from the range in the overview.

        Args:
            label_name: The name of the label to remove.
        """
        dropdown = self._open_labels_dropdown()
        item = dropdown.locator(".pluto-list__item").filter(has_text=label_name).first
        item.click(timeout=5000)
        self.layout.press_escape()
        dropdown.wait_for(state="hidden", timeout=5000)

    def get_labels_in_overview(self) -> list[str]:
        """Get the labels currently attached to the range in the overview.

        Returns:
            A list of label names.
        """
        labels_row = self.layout.page.get_by_text("Labels", exact=True).locator("..")
        label_chips = labels_row.locator(".pluto-tag")
        labels = []
        for i in range(label_chips.count()):
            text = label_chips.nth(i).inner_text()
            if text:
                labels.append(text)
        return labels

    def rename_from_overview(self, new_name: str) -> None:
        """Rename the range from the overview name field.

        Args:
            new_name: The new name for the range.
        """
        name_input = self.layout.page.locator("input[placeholder='Name']").first
        name_input.wait_for(state="visible", timeout=5000)
        name_input.click()
        name_input.fill(new_name)
        name_input.blur()
        expect(name_input).to_have_value(new_name, timeout=5000)

    def copy_python_code_from_overview(self) -> None:
        """Click the Python code copy button in the range overview."""
        python_btn = self.layout.page.locator("button:has(svg.pluto-icon--python)")
        python_btn.click(timeout=5000)

    def copy_typescript_code_from_overview(self) -> None:
        """Click the TypeScript code copy button in the range overview."""
        ts_btn = self.layout.page.locator("button:has(svg.pluto-icon--typescript)")
        ts_btn.click(timeout=5000)

    def copy_link_from_overview(self) -> None:
        """Click the copy link button in the range overview."""
        link_btn = self.layout.page.locator("button:has(svg.pluto-icon--link)")
        link_btn.click(timeout=5000)

    def open_csv_download_modal(self) -> None:
        """Click the CSV download button in the range overview and wait for modal."""
        csv_btn = self.layout.page.locator("button:has(svg.pluto-icon--csv)")
        csv_btn.click(timeout=5000)
        self.layout.page.get_by_text("Download data for").wait_for(
            state="visible", timeout=5000
        )

    def close_csv_download_modal(self) -> None:
        """Close the CSV download modal."""
        close_btn = self.layout.page.locator("button:has(svg.pluto-icon--close)").first
        close_btn.click(timeout=2000)
        self.layout.page.get_by_text("Download data for").wait_for(
            state="hidden", timeout=5000
        )

    def download_csv(self, range_name: str, channel: str) -> str:
        """Download CSV data for a range with specified channel.

        Args:
            range_name: The name of the range (used for file naming).
            channel: The channel name to select for download.

        Returns:
            The CSV file contents as a string.
        """
        self.notifications.close_all()
        self.open_csv_download_modal()

        channels_dropdown = self.layout.page.get_by_text("Select channels to download")
        channels_dropdown.click(timeout=5000)
        search_input = self.layout.page.locator("input[placeholder*='Search']")
        search_input.fill(channel)
        self.layout.select_from_dropdown(channel)
        self.layout.press_escape()

        download_button = self.layout.page.get_by_role("button", name="Download").last
        self.layout.page.evaluate("delete window.showSaveFilePicker")

        with self.layout.page.expect_download() as download_info:
            download_button.click()

        download = download_info.value
        save_path = get_results_path(f"{range_name}.csv")
        download.save_as(save_path)
        with open(save_path, "r") as f:
            return f.read()

    # ── Metadata ──────────────────────────────────────────────────────────

    METADATA_SECTION_SELECTOR = ".console-range__metadata"
    METADATA_ITEM_SELECTOR = ".console-metadata__list-item"
    METADATA_CREATE_SELECTOR = ".console-metadata__list-item.console--create"
    METADATA_DELETE_SELECTOR = ".console-metadata__delete"

    def _get_metadata_section(self) -> Locator:
        """Get the Metadata section in the range overview."""
        return self.layout.page.locator(self.METADATA_SECTION_SELECTOR)

    def get_metadata_item(self, key: str) -> Locator:
        """Get a metadata list item by key name.

        Args:
            key: The metadata key to find.

        Returns:
            The Locator for the metadata item row.
        """
        section = self._get_metadata_section()
        return (
            section.locator(f"{self.METADATA_ITEM_SELECTOR}:not(.console--create)")
            .filter(has_text=key)
            .first
        )

    def metadata_exists(self, key: str) -> bool:
        """Check if a metadata entry with the given key exists.

        Args:
            key: The metadata key to check.

        Returns:
            True if the metadata key exists in the overview.
        """
        return self.layout.locator_exists(self.get_metadata_item(key))

    def wait_for_metadata_removed(self, key: str) -> None:
        """Wait for a metadata entry to be removed.

        Args:
            key: The metadata key to wait for removal.
        """
        self.get_metadata_item(key).wait_for(state="hidden", timeout=5000)

    def set_metadata(self, key: str, value: str) -> None:
        """Add a new metadata key-value pair.

        Args:
            key: The metadata key.
            value: The metadata value.
        """
        section = self._get_metadata_section()
        add_btn = section.locator("button:has(svg.pluto-icon--add)")
        add_btn.click(timeout=5000)

        create_form = section.locator(
            f"{self.METADATA_CREATE_SELECTOR}:not(.pluto--hidden)"
        )
        create_form.wait_for(state="visible", timeout=5000)

        key_input = create_form.locator("input[placeholder='Key']")
        key_input.fill(key)

        value_input = create_form.locator("input[placeholder='Value']")
        value_input.fill(value)

        self.layout.press_key("Enter")
        self.get_metadata_item(key).wait_for(state="visible", timeout=5000)

    def get_metadata_value(self, key: str) -> str:
        """Get the current value of a metadata entry.

        Args:
            key: The metadata key.

        Returns:
            The value string.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        value_input = item.locator("input[placeholder='Value']")
        return value_input.input_value()

    def update_metadata_value(self, key: str, new_value: str) -> None:
        """Update the value of an existing metadata entry.

        The value input uses onlyChangeOnBlur, so we must blur it to trigger
        the form onChange and auto-save. For existing items the key is rendered
        as plain Text (not an input), so we click the section header to blur.

        Args:
            key: The metadata key to update.
            new_value: The new value to set.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        value_input = item.locator("input[placeholder='Value']")
        value_input.click()
        value_input.fill(new_value)
        # Press Tab to blur the value input, triggering onlyChangeOnBlur + auto-save
        self.layout.page.keyboard.press("Tab")
        expect(value_input).to_have_value(new_value, timeout=5000)

    def copy_metadata_value(self, key: str) -> None:
        """Click the copy button on a metadata value.

        Args:
            key: The metadata key whose value to copy.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        copy_btn = item.locator("button:has(svg.pluto-icon--copy)")
        copy_btn.click(timeout=5000)

    def delete_metadata(self, key: str) -> None:
        """Delete a metadata entry by hovering and clicking the delete button.

        Args:
            key: The metadata key to delete.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        item.hover()
        delete_btn = item.locator(self.METADATA_DELETE_SELECTOR)
        delete_btn.click(timeout=5000)
        item.wait_for(state="hidden", timeout=5000)

    def open_metadata_link(self, key: str) -> None:
        """Click the external link button on a metadata value that contains a URL.

        Args:
            key: The metadata key whose link to open.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        link_btn = item.locator("a:has(svg.pluto-icon--link-external)")
        link_btn.wait_for(state="visible", timeout=10000)
        link_btn.click(timeout=5000)

    # ── Child Ranges ─────────────────────────────────────────────────────

    def _get_child_ranges_section(self) -> Locator:
        """Get the Child Ranges section in the overview."""
        return (
            self.layout.page.get_by_text("Child Ranges", exact=True)
            .locator("..")
            .locator("..")
        )

    def get_child_range_item(self, name: str) -> Locator:
        """Get a child range item from the Child Ranges section by name.

        Args:
            name: The name of the child range.

        Returns:
            The Locator for the child range item.
        """
        section = self._get_child_ranges_section()
        return section.locator(".console-range__list-item").filter(has_text=name).first

    def click_child_range(self, name: str) -> None:
        """Click on a child range to navigate to its overview.

        Args:
            name: The name of the child range to click.
        """
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()

    def create_child_range_from_overview(self) -> None:
        """Click the Add button in the Child Ranges section to create a new child range."""
        self.notifications.close_all()
        section = self._get_child_ranges_section()
        add_btn = section.locator("button:has(svg.pluto-icon--add)")
        add_btn.click(timeout=5000)
        modal = self.layout.page.locator(self.CREATE_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)

    def set_child_range_stage(self, name: str, stage: str) -> None:
        """Change the stage of a child range in the Child Ranges section.

        Args:
            name: The name of the child range.
            stage: The stage to set ("To Do", "In Progress", "Completed").
        """
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        stage_button = (
            item.locator("button")
            .filter(has=self.layout.page.locator("svg.pluto-icon--to-do"))
            .or_(
                item.locator("button").filter(
                    has=self.layout.page.locator("svg.pluto-icon--in-progress")
                )
            )
            .or_(
                item.locator("button").filter(
                    has=self.layout.page.locator("svg.pluto-icon--completed")
                )
            )
            .first
        )
        self._pick_stage_from_dropdown(stage_button, stage)

    def favorite_child_range(self, name: str) -> None:
        """Favorite a child range from the Child Ranges section via context menu.

        Args:
            name: The name of the child range to favorite.
        """
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.open_on(item)
        add_btn = self.layout.page.get_by_text("Add to favorites", exact=True)
        remove_btn = self.layout.page.get_by_text("Remove from favorites", exact=True)
        add_btn.or_(remove_btn).wait_for(state="visible", timeout=2000)
        if remove_btn.is_visible():
            self.ctx_menu.click_option("Remove from favorites")
            self.ctx_menu.open_on(item)
        self.ctx_menu.click_option("Add to favorites")

    def unfavorite_child_range(self, name: str) -> None:
        """Unfavorite a child range from the Child Ranges section via context menu.

        Args:
            name: The name of the child range to unfavorite.
        """
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Remove from favorites")
        self.wait_for_removed_from_toolbar(name)

    def child_range_exists(self, name: str) -> bool:
        """Check if a child range exists in the Child Ranges section."""
        return self.layout.locator_exists(self.get_child_range_item(name))

    def wait_for_child_range_removed(self, name: str) -> None:
        """Wait for a child range to be removed from the Child Ranges section.

        Args:
            name: The name of the child range.
        """
        self.get_child_range_item(name).wait_for(state="hidden", timeout=5000)

    def rename_child_range(self, name: str, new_name: str) -> None:
        """Rename a child range via context menu modal dialog."""
        self.layout.rename_with_modal(self.get_child_range_item(name), new_name)

    def copy_link_from_child_range(self, name: str) -> None:
        """Copy link to a child range via context menu.

        Args:
            name: The name of the child range.
        """
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Copy link")

    def delete_child_range(self, name: str) -> None:
        """Delete a child range via context menu with confirmation."""
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.layout.delete_with_confirmation(item)
        item.wait_for(state="hidden", timeout=5000)

    def _deselect_all_child_ranges(self) -> None:
        """Deselect all child ranges by dispatching click on their checkbox labels."""
        self.layout.deselect_all_items(
            self._get_child_ranges_section(), ".console-range__list-item"
        )

    def _select_child_ranges(self, names: list[str]) -> Locator:
        """Select multiple child ranges via their checkbox labels."""
        return self.layout.select_items(names, self.get_child_range_item)

    def delete_child_ranges(self, names: list[str]) -> None:
        """Delete multiple child ranges via multi-select and context menu."""
        if not names:
            return
        last_item = self._select_child_ranges(names)
        self.ctx_menu.action(last_item, "Delete")
        self.layout.confirm_delete()
        for name in names:
            self.wait_for_child_range_removed(name)

    def favorite_child_ranges(self, names: list[str]) -> None:
        """Favorite multiple child ranges via multi-select and context menu.

        Args:
            names: The names of the child ranges to favorite.
        """
        if not names:
            return

        last_item = self._select_child_ranges(names)
        self.ctx_menu.action(last_item, "Add to favorites")
        self._deselect_all_child_ranges()

    def unfavorite_child_ranges(self, names: list[str]) -> None:
        """Unfavorite multiple child ranges via multi-select and context menu.

        Args:
            names: The names of the child ranges to unfavorite.
        """
        if not names:
            return

        last_item = self._select_child_ranges(names)
        self.ctx_menu.action(last_item, "Remove from favorites")
        self._deselect_all_child_ranges()

        for name in names:
            self.wait_for_removed_from_toolbar(name)

    # ── Explorer Multi-Select & Context Menu ─────────────────────────────

    def _deselect_all_explorer_ranges(self) -> None:
        """Deselect all explorer ranges by dispatching click on their checkbox labels."""
        self.layout.deselect_all_items(self.layout.page, self.EXPLORER_ITEM_SELECTOR)

    def _select_explorer_ranges(self, names: list[str]) -> Locator:
        """Select multiple explorer ranges via their checkbox labels."""
        return self.layout.select_items(names, self.get_explorer_item)

    def unfavorite_from_explorer(self, name: str) -> None:
        """Remove a range from favorites via context menu in the explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Remove from favorites")
        self.wait_for_removed_from_toolbar(name)

    def favorite_explorer_ranges(self, names: list[str]) -> None:
        """Favorite multiple ranges via multi-select context menu in the explorer.

        Args:
            names: The names of the ranges to favorite.
        """
        if not names:
            return
        last_item = self._select_explorer_ranges(names)
        self.ctx_menu.action(last_item, "Add to favorites")
        self._deselect_all_explorer_ranges()

    def unfavorite_explorer_ranges(self, names: list[str]) -> None:
        """Unfavorite multiple ranges via multi-select context menu in the explorer.

        Args:
            names: The names of the ranges to unfavorite.
        """
        if not names:
            return
        last_item = self._select_explorer_ranges(names)
        self.ctx_menu.action(last_item, "Remove from favorites")
        self._deselect_all_explorer_ranges()
        for name in names:
            self.wait_for_removed_from_toolbar(name)

    def delete_explorer_ranges(self, names: list[str]) -> None:
        """Delete multiple ranges via multi-select context menu in the explorer."""
        if not names:
            return
        last_item = self._select_explorer_ranges(names)
        self.ctx_menu.action(last_item, "Delete")
        self.layout.confirm_delete()
        for name in names:
            self.wait_for_removed_from_explorer(name)

    def copy_link_from_explorer(self, name: str) -> None:
        """Copy link to a range via context menu in the explorer.

        Args:
            name: The name of the range.
        """
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Copy link")

    def create_child_range_from_explorer(
        self, parent_name: str, child_name: str
    ) -> None:
        """Create a child range via context menu in the explorer.

        Args:
            parent_name: The name of the parent range.
            child_name: The name for the new child range.
        """
        item = self.get_explorer_item(parent_name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Create child range")
        modal = self.layout.page.locator(self.CREATE_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        self._fill_create_modal(child_name)

    # ── Explorer Search & Filter ──────────────────────────────────────────

    SEARCH_INPUT_PLACEHOLDER = "Search ranges..."

    def enable_explorer_editing(self) -> None:
        """Enable editing mode in the explorer to show search/filter controls."""
        search_input = self.layout.page.locator(
            f"input[placeholder='{self.SEARCH_INPUT_PLACEHOLDER}']"
        )
        if search_input.is_visible():
            return
        edit_button = (
            self.layout.page.locator("button")
            .filter(has=self.layout.page.locator("svg.pluto-icon--edit"))
            .first
        )
        edit_button.click()
        search_input.wait_for(state="visible", timeout=5000)

    def search_explorer(self, term: str) -> None:
        """Type a search term in the explorer search input.

        Args:
            term: The search string to type.
        """
        self.enable_explorer_editing()
        search_input = self.layout.page.get_by_placeholder(
            self.SEARCH_INPUT_PLACEHOLDER
        )
        search_input.fill(term)
        search_input.dispatch_event(
            "input",
            {"bubbles": True, "data": term, "inputType": "insertText"},
        )

    def clear_explorer_search(self) -> None:
        """Clear the explorer search input."""
        self.search_explorer("")

    def open_explorer_label_filter(self) -> Locator:
        """Open the label filter dropdown in the explorer.

        Returns:
            Locator for the visible filter dialog.
        """
        self.enable_explorer_editing()
        filter_button = (
            self.layout.page.locator("button")
            .filter(has=self.layout.page.locator("svg.pluto-icon--filter"))
            .first
        )
        filter_button.click()
        dialog = self.layout.page.locator(".pluto-dialog__dialog.pluto--visible")
        dialog.wait_for(state="visible", timeout=5000)
        return dialog

    def select_explorer_label_filter(self, label_name: str) -> None:
        """Select a label in the explorer's label filter dropdown.

        The filter is a two-level dialog:
        1. Filter button → first dialog with "Select labels" trigger
        2. "Select labels" → second dialog with label list

        Args:
            label_name: The name of the label to select.
        """
        filter_dialog = self.open_explorer_label_filter()
        select_labels_trigger = filter_dialog.get_by_text("Select labels")
        select_labels_trigger.click()
        label_dialog = self.layout.page.locator(".pluto-select__dialog.pluto--visible")
        label_dialog.wait_for(state="visible", timeout=5000)
        item = (
            label_dialog.locator(".pluto-list__item").filter(has_text=label_name).first
        )
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.layout.press_escape()
        self.layout.press_escape()

    def clear_explorer_label_filter(self, label_name: str) -> None:
        """Remove a label from the active filter by clicking its chip close button.

        Args:
            label_name: The name of the label chip to remove.
        """
        tag = (
            self.layout.page.locator(".pluto-tag:has(button)")
            .filter(has_text=label_name)
            .first
        )
        tag.wait_for(state="visible", timeout=5000)
        close_btn = tag.locator("button")
        close_btn.click()
        tag.wait_for(state="hidden", timeout=5000)

    # ── Range Label Operations ─────────────────────────────────────────────

    def get_label_in_toolbar(self, range_name: str, label_name: str) -> Locator:
        """Get a label tag within a range item in the toolbar."""
        range_item = self.get_toolbar_item(range_name)
        return range_item.locator(".pluto-tag").filter(has_text=label_name).first

    def label_exists_in_toolbar(self, range_name: str, label_name: str) -> bool:
        """Check if a label exists on a range in the toolbar."""
        self.show_toolbar()
        return self.layout.locator_exists(
            self.get_label_in_toolbar(range_name, label_name)
        )

    def wait_for_label_removed_from_toolbar(
        self, range_name: str, label_name: str
    ) -> None:
        """Wait until a label is removed from a range in the toolbar."""
        self.show_toolbar()
        self.layout.wait_for_hidden(self.get_label_in_toolbar(range_name, label_name))

    def get_label_color_in_toolbar(
        self, range_name: str, label_name: str
    ) -> str | None:
        """Get the color of a label's icon in the range toolbar."""
        self.show_toolbar()
        label = self.get_label_in_toolbar(range_name, label_name)
        if label.count() == 0:
            return None
        icon = label.locator("svg").first
        if icon.count() == 0:
            return None
        color = icon.get_attribute("color")
        if color is None:
            return None
        return rgb_to_hex(color)

    def get_all_labels_in_toolbar(self, range_name: str) -> list[str]:
        """Get all labels currently visible for a range in the toolbar.

        Args:
            range_name: The name of the range to check.

        Returns:
            List of label names currently displayed in the toolbar for this range.
        """
        self.show_toolbar()
        range_item = self.get_toolbar_item(range_name)
        if not self.layout.locator_exists(range_item):
            return []

        label_tags = range_item.locator(".pluto-tag")
        label_count = label_tags.count()

        labels = []
        for i in range(label_count):
            label_text = label_tags.nth(i).text_content()
            if label_text:
                labels.append(label_text.strip())

        return labels

    # ── Range Snapshot Operations ───────────────────────────────────────────

    def get_snapshot_item(self, name: str) -> Locator:
        """Get a snapshot item locator from the Snapshots section by name.

        Args:
            name: The name of the snapshot to find.

        Returns:
            Locator for the snapshot item.
        """
        return self.layout.page.locator(".console-snapshots__list-item").filter(
            has_text=name
        )

    def snapshot_exists_in_overview(self, name: str) -> bool:
        """Check if a snapshot exists in the Snapshots section of the overview."""
        return self.layout.locator_exists(self.get_snapshot_item(name))

    def open_snapshot_from_overview(self, name: str) -> None:
        """Open a snapshot from the Snapshots section in the range overview.

        Args:
            name: The name of the snapshot to open.
        """
        item = self.get_snapshot_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()

    def get_snapshot_names_in_overview(self) -> list[str]:
        """Get all snapshot names in the Snapshots section.

        Returns:
            List of snapshot names.
        """
        items = self.layout.page.locator(".console-snapshots__list-item")
        return [items.nth(i).inner_text().strip() for i in range(items.count())]

    def delete_snapshot_from_overview(self, name: str) -> None:
        """Delete a snapshot from the Snapshots section in the range overview.

        Args:
            name: The name of the snapshot to delete.
        """
        item = self.get_snapshot_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.locator(".console-snapshots__delete").click()
        self.layout.confirm_delete()

    def snapshot_to_active_range(self, name: str, range_name: str) -> None:
        """Snapshot a page or task to the active range via context menu.

        Searches the workspace tree first (for schematics, line plots, etc.),
        then falls back to the task toolbar (for hardware tasks).

        Args:
            name: Name of the page or task to snapshot.
            range_name: Name of the active range (for menu text matching).
        """
        self.layout.show_resource_toolbar("workspace")
        self.tree.expand_root("workspace:")
        page_item = (
            self.layout.page.locator(".pluto-tree__item").filter(has_text=name).first
        )
        if self.layout.locator_exists(page_item):
            self.ctx_menu.action(page_item, f"Snapshot to {range_name}")
        else:
            self.layout.show_resource_toolbar("task")
            task_item = (
                self.layout.page.locator(".pluto-list__item")
                .filter(has_text=name)
                .first
            )
            task_item.wait_for(state="visible", timeout=5000)
            self.ctx_menu.action(task_item, f"Snapshot to {range_name}")
        self.layout.close_left_toolbar()

    def wait_for_snapshot_removed(self, name: str) -> None:
        """Wait for a snapshot to be removed from the Snapshots section.

        Args:
            name: The name of the snapshot that should be removed.
        """
        self.get_snapshot_item(name).wait_for(state="hidden", timeout=10000)
