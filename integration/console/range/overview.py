#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator, expect
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.layout import LayoutClient
from console.range.explorer import Explorer
from console.range.surface import (
    CREATE_MODAL_SELECTOR,
    FAVORITE_ACTIONS,
    UNFAVORITE_ACTIONS,
    Surface,
)
from console.range.toolbar import Toolbar
from framework.run_dir import resolve_results_path


class Overview(Surface):
    """A range's overview page: times, stage, labels, metadata, child ranges,
    and snapshots."""

    METADATA_SECTION_SELECTOR = ".console-range__metadata"
    METADATA_ITEM_SELECTOR = ".console-metadata__list-item"
    METADATA_CREATE_SELECTOR = ".console-metadata__list-item.console--create"
    METADATA_DELETE_SELECTOR = ".console-metadata__delete"

    def __init__(self, layout: LayoutClient, toolbar: Toolbar, explorer: Explorer):
        super().__init__(layout)
        self.toolbar = toolbar
        self.explorer = explorer

    def open(self, name: str) -> None:
        """Open the range overview/details page from the explorer.

        Clicking an explorer row places the range's overview layout, which opens
        a new overview tab or focuses an existing one. When an overview tab for
        the range is already open, the row click does not reliably switch the
        active tab to it (the explorer tab can stay focused), leaving the
        overview's Name field hidden. Focus an already-open tab directly when
        present, fall back to the row click otherwise, and retry until the
        overview for this range is showing.
        """
        if self.is_showing(name):
            return
        item = self.explorer.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        for attempt in range(3):
            tab = self.layout.get_tab(name)
            if tab.count() > 0:
                tab.click()
            else:
                item.click()
            try:
                self.wait_for(name)
                return
            except PlaywrightTimeoutError:
                if attempt == 2:
                    raise

    def wait_for(self, name: str) -> None:
        """Wait for the range overview to show a specific range."""
        name_input = self.layout.page.locator("input[placeholder='Name']:visible").first
        name_input.wait_for(state="visible", timeout=5000)
        expect(name_input).to_have_value(name, timeout=5000)

    def is_showing(self, name: str) -> bool:
        """Check if the range overview is showing a specific range.

        :param name: The name of the range to check for.
        :returns: True if the overview shows the range name in the header.
        """
        header = self.layout.page.locator("input[placeholder='Name']:visible").first
        if not header.is_visible():
            return False
        return header.input_value() == name

    def navigate_to_parent(self, parent_name: str) -> None:
        """Navigate to parent range from current range overview.

        :param parent_name: The name of the parent range to navigate to.
        """
        parent_button = self.layout.page.get_by_role("button").filter(
            has_text=parent_name
        )
        parent_button.click(timeout=5000)

    # ── Times and stage ──────────────────────────────────────────────────

    def set_start_time(
        self,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Set the start time in the range overview."""
        self._set_time(0, year, month, day, hour, minute, second)

    def set_end_time(
        self,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Set the end time in the range overview."""
        self._set_time(1, year, month, day, hour, minute, second)

    def _set_time(
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
        """Select a value from a time list by clicking the item with matching
        id."""
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

    def set_stage(self, stage: str) -> None:
        """Set the stage in the range overview.

        :param stage: The stage to set ("To do", "In progress", "Completed").
        """
        stage_button = (
            self.layout.page.locator("button")
            .filter(has_text="To do")
            .or_(self.layout.page.locator("button").filter(has_text="In progress"))
            .or_(self.layout.page.locator("button").filter(has_text="Completed"))
            .first
        )
        self._pick_stage_from_dropdown(stage_button, stage)

    # ── Labels ───────────────────────────────────────────────────────────

    def _open_labels_dropdown(self) -> Locator:
        """Open the labels dropdown in the range overview and return the
        dialog."""
        labels_row = self.layout.page.get_by_text("Labels", exact=True).locator("..")
        # The add button only renders once a label is set. Before that, the
        # placeholder is the trigger.
        add_button = labels_row.locator("button").last
        placeholder = labels_row.locator(".pluto-select-multiple-trigger-placeholder")
        trigger = add_button.or_(placeholder).first
        trigger.wait_for(state="visible", timeout=2000)
        trigger.click()
        dropdown = self.layout.dialog
        dropdown.wait_for(state="visible", timeout=5000)
        return dropdown

    def add_label(self, label_name: str) -> None:
        """Add a label to the range in the overview.

        :param label_name: The name of the label to add.
        """
        dropdown = self._open_labels_dropdown()
        item = dropdown.get_by_role("option").filter(has_text=label_name).first
        try:
            item.wait_for(state="visible", timeout=5000)
            item.click(timeout=2000)
        except PlaywrightTimeoutError as e:
            available_labels = dropdown.get_by_role("option").all_text_contents()
            raise PlaywrightTimeoutError(
                f"Label '{label_name}' not found in dropdown. "
                f"Available: {available_labels}"
            ) from e
        self.layout.press_escape()
        dropdown.wait_for(state="hidden", timeout=5000)

    def remove_label(self, label_name: str) -> None:
        """Remove a label from the range in the overview.

        :param label_name: The name of the label to remove.
        """
        dropdown = self._open_labels_dropdown()
        item = dropdown.get_by_role("option").filter(has_text=label_name).first
        item.click(timeout=5000)
        self.layout.press_escape()
        dropdown.wait_for(state="hidden", timeout=5000)

    def get_labels(self) -> list[str]:
        """Get the labels currently attached to the range in the overview.

        :returns: A list of label names.
        """
        labels_row = self.layout.page.get_by_text("Labels", exact=True).locator("..")
        label_chips = labels_row.locator(".pluto-tag")
        labels = []
        for i in range(label_chips.count()):
            text = label_chips.nth(i).inner_text()
            if text:
                labels.append(text)
        return labels

    # ── Header actions ───────────────────────────────────────────────────

    def rename(self, new_name: str) -> None:
        """Rename the range from the overview name field.

        :param new_name: The new name for the range.
        """
        name_input = self.layout.page.locator("input[placeholder='Name']:visible").first
        name_input.wait_for(state="visible", timeout=5000)
        name_input.click()
        name_input.fill(new_name)
        name_input.blur()
        expect(name_input).to_have_value(new_name, timeout=5000)

    def copy_python_code(self) -> None:
        """Click the Python code copy button in the range overview."""
        python_btn = self.layout.page.locator("button:has(svg.pluto-icon--python)")
        python_btn.wait_for(state="visible", timeout=5000)
        python_btn.click(timeout=5000)

    def copy_typescript_code(self) -> None:
        """Click the TypeScript code copy button in the range overview."""
        ts_btn = self.layout.page.locator("button:has(svg.pluto-icon--typescript)")
        ts_btn.wait_for(state="visible", timeout=5000)
        ts_btn.click(timeout=5000)

    def copy_link(self) -> None:
        """Click the copy link button in the range overview."""
        link_btn = self.layout.page.locator("button:has(svg.pluto-icon--link)")
        link_btn.wait_for(state="visible", timeout=5000)
        link_btn.click(timeout=5000)

    def open_csv_download_modal(self) -> None:
        """Click the CSV download button in the range overview and wait for
        modal."""
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

        :param range_name: The name of the range (used for file naming).
        :param channel: The channel name to select for download.
        :returns: The CSV file contents as a string.
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

        with self.layout.page.expect_download() as download_info:
            download_button.click()

        download = download_info.value
        save_path = resolve_results_path(f"{range_name}.csv")
        download.save_as(save_path)
        with open(save_path, "r", encoding="utf-8") as f:
            return f.read()

    # ── Metadata ─────────────────────────────────────────────────────────

    def _get_metadata_section(self) -> Locator:
        """Get the Metadata section in the range overview."""
        return self.layout.page.locator(self.METADATA_SECTION_SELECTOR)

    def get_metadata_item(self, key: str) -> Locator:
        """Get a metadata list item by key name.

        :param key: The metadata key to find.
        :returns: The Locator for the metadata item row.
        """
        section = self._get_metadata_section()
        return (
            section.locator(f"{self.METADATA_ITEM_SELECTOR}:not(.console--create)")
            .filter(has_text=key)
            .first
        )

    def metadata_exists(self, key: str) -> bool:
        """Check if a metadata entry with the given key exists.

        :param key: The metadata key to check.
        :returns: True if the metadata key exists in the overview.
        """
        return self.layout.locator_exists(self.get_metadata_item(key))

    def wait_for_metadata_removed(self, key: str) -> None:
        """Wait for a metadata entry to be removed.

        :param key: The metadata key to wait for removal.
        """
        self.get_metadata_item(key).wait_for(state="hidden", timeout=5000)

    def set_metadata(self, key: str, value: str) -> None:
        """Add a new metadata key-value pair.

        :param key: The metadata key.
        :param value: The metadata value.
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

        :param key: The metadata key.
        :returns: The value string.
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

        :param key: The metadata key to update.
        :param new_value: The new value to set.
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

        :param key: The metadata key whose value to copy.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        copy_btn = item.locator("button:has(svg.pluto-icon--copy)")
        copy_btn.click(timeout=5000)

    def delete_metadata(self, key: str) -> None:
        """Delete a metadata entry by hovering and clicking the delete button.

        :param key: The metadata key to delete.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        item.hover()
        delete_btn = item.locator(self.METADATA_DELETE_SELECTOR)
        delete_btn.click(timeout=5000)
        item.wait_for(state="hidden", timeout=5000)

    def open_metadata_link(self, key: str) -> None:
        """Click the external link button on a metadata value that contains a
        URL.

        :param key: The metadata key whose link to open.
        """
        item = self.get_metadata_item(key)
        item.wait_for(state="visible", timeout=5000)
        link_btn = item.get_by_role("link")
        link_btn.wait_for(state="visible", timeout=10000)
        link_btn.click(timeout=5000)

    # ── Child ranges ─────────────────────────────────────────────────────

    def _get_child_ranges_section(self) -> Locator:
        """Get the Child Ranges section in the overview."""
        return (
            self.layout.page.get_by_text("Child ranges", exact=True)
            .locator("..")
            .locator("..")
        )

    def get_child_range(self, name: str) -> Locator:
        """Get a child range item from the Child Ranges section by name.

        :param name: The name of the child range.
        :returns: The Locator for the child range item.
        """
        section = self._get_child_ranges_section()
        return section.locator(".console-range__list-item").filter(has_text=name).first

    def wait_for_child_ranges(self, names: list[str], parent_name: str) -> None:
        """Wait for child ranges to appear, reopening the overview as a
        fallback.

        Child ranges created via the API propagate to the console through a
        reactive subscription. On slower CI machines this can lag, so if the
        initial wait times out we close and reopen the overview to trigger a
        fresh retrieval.
        """
        try:
            for name in names:
                self.get_child_range(name).wait_for(state="visible", timeout=10000)
        except PlaywrightTimeoutError:
            self.layout.close_tab(parent_name)
            self.explorer.open()
            self.open(parent_name)
            self.wait_for(parent_name)
            for name in names:
                self.get_child_range(name).wait_for(state="visible", timeout=10000)

    def click_child_range(self, name: str) -> None:
        """Click on a child range to navigate to its overview.

        :param name: The name of the child range to click.
        """
        item = self.get_child_range(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()

    def create_child_range(self, name: str) -> None:
        """Create a child range from the Child Ranges section of the overview.

        :param name: The name for the new child range.
        """
        self.notifications.close_all()
        section = self._get_child_ranges_section()
        add_btn = section.locator("button:has(svg.pluto-icon--add)")
        add_btn.click(timeout=5000)
        modal = self.layout.page.locator(CREATE_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        self.fill_create_modal(name)

    def set_child_range_stage(self, name: str, stage: str) -> None:
        """Change the stage of a child range in the Child Ranges section.

        :param name: The name of the child range.
        :param stage: The stage to set ("To do", "In progress", "Completed").
        """
        item = self.get_child_range(name)
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
        """Favorite a child range from the Child Ranges section via context
        menu.

        :param name: The name of the child range to favorite.
        """
        item = self.get_child_range(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.open_on(item)
        menu = self.layout.page.get_by_role("menu")
        add_btn = self._any_text_locator(menu, FAVORITE_ACTIONS)
        remove_btn = self._any_text_locator(menu, UNFAVORITE_ACTIONS)
        add_btn.or_(remove_btn).wait_for(state="visible", timeout=2000)
        if remove_btn.is_visible():
            self._click_visible_option(UNFAVORITE_ACTIONS)
            self.ctx_menu.open_on(item)
        self._click_visible_option(FAVORITE_ACTIONS)

    def unfavorite_child_range(self, name: str) -> None:
        """Unfavorite a child range from the Child Ranges section via context
        menu.

        :param name: The name of the child range to unfavorite.
        """
        item = self.get_child_range(name)
        item.wait_for(state="visible", timeout=5000)
        self._ctx_action_any(item, UNFAVORITE_ACTIONS)
        self.toolbar.wait_for_removed(name)

    def child_range_exists(self, name: str) -> bool:
        """Check if a child range exists in the Child Ranges section."""
        return self.layout.locator_exists(self.get_child_range(name))

    def wait_for_child_range_removed(self, name: str) -> None:
        """Wait for a child range to be removed from the Child Ranges section.

        :param name: The name of the child range.
        """
        self.get_child_range(name).wait_for(state="hidden", timeout=5000)

    def rename_child_range(self, name: str, new_name: str) -> None:
        """Rename a child range via context menu modal dialog."""
        self.layout.rename_with_modal(self.get_child_range(name), new_name)

    def copy_child_range_link(self, name: str) -> None:
        """Copy link to a child range via context menu.

        :param name: The name of the child range.
        """
        item = self.get_child_range(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Copy link")

    def delete_child_range(self, name: str) -> None:
        """Delete a child range via context menu with confirmation."""
        item = self.get_child_range(name)
        item.wait_for(state="visible", timeout=5000)
        self.layout.delete_with_confirmation(item)
        item.wait_for(state="hidden", timeout=5000)

    def _deselect_all_child_ranges(self) -> None:
        """Deselect all child ranges by dispatching click on their checkbox
        labels."""
        self.layout.deselect_all_items(
            self._get_child_ranges_section(), ".console-range__list-item"
        )

    def _select_child_ranges(self, names: list[str]) -> Locator:
        """Select multiple child ranges via their checkbox labels."""
        return self.layout.select_items(names, self.get_child_range)

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

        :param names: The names of the child ranges to favorite.
        """
        if not names:
            return
        last_item = self._select_child_ranges(names)
        self._ctx_action_any(last_item, FAVORITE_ACTIONS)
        self._deselect_all_child_ranges()

    def unfavorite_child_ranges(self, names: list[str]) -> None:
        """Unfavorite multiple child ranges via multi-select and context menu.

        :param names: The names of the child ranges to unfavorite.
        """
        if not names:
            return
        last_item = self._select_child_ranges(names)
        self._ctx_action_any(last_item, UNFAVORITE_ACTIONS)
        self._deselect_all_child_ranges()
        for name in names:
            self.toolbar.wait_for_removed(name)

    # ── Snapshots ────────────────────────────────────────────────────────

    def get_snapshot(self, name: str) -> Locator:
        """Get a snapshot item locator from the Snapshots section by name.

        :param name: The name of the snapshot to find.
        :returns: Locator for the snapshot item.
        """
        return self.layout.page.locator(".console-snapshots__list-item").filter(
            has_text=name
        )

    def snapshot_exists(self, name: str) -> bool:
        """Check if a snapshot exists in the Snapshots section of the
        overview."""
        return self.layout.locator_exists(self.get_snapshot(name))

    def open_snapshot(self, name: str) -> None:
        """Open a snapshot from the Snapshots section in the range overview.

        :param name: The name of the snapshot to open.
        """
        item = self.get_snapshot(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()

    def get_snapshot_names(self) -> list[str]:
        """Get all snapshot names in the Snapshots section.

        :returns: List of snapshot names.
        """
        items = self.layout.page.locator(".console-snapshots__list-item")
        return [items.nth(i).inner_text().strip() for i in range(items.count())]

    def delete_snapshot(self, name: str) -> None:
        """Delete a snapshot from the Snapshots section in the range overview.

        :param name: The name of the snapshot to delete.
        """
        item = self.get_snapshot(name)
        item.wait_for(state="visible", timeout=5000)
        item.get_by_role("button", name=f"Delete {name}", exact=True).click()
        self.layout.confirm_delete()

    def wait_for_snapshot_removed(self, name: str) -> None:
        """Wait for a snapshot to be removed from the Snapshots section.

        :param name: The name of the snapshot that should be removed.
        """
        self.get_snapshot(name).wait_for(state="hidden", timeout=5000)
