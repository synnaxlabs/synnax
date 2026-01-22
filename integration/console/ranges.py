#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING

from playwright.sync_api import Locator, Page

from framework.utils import get_results_path, rgb_to_hex

if TYPE_CHECKING:
    from .console import Console


class RangesClient:
    """Console ranges client for managing ranges via the UI.

    The ranges toolbar shows only favorited ranges.
    The Range Explorer shows all persisted ranges.
    """

    TOOLBAR_ITEM_SELECTOR = ".console-range-list-item"
    EXPLORER_ITEM_SELECTOR = ".console-range__list-item"
    CREATE_MODAL_SELECTOR = ".console-range-create-layout"
    NAME_INPUT_PLACEHOLDER = "Range Name"

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console

    def show_toolbar(self) -> None:
        """Show the ranges toolbar in the left sidebar (favorites only)."""
        toolbar_header = self.page.get_by_text("Ranges", exact=True).first
        if toolbar_header.is_visible():
            return
        self.page.keyboard.press("r")
        toolbar_header.wait_for(state="visible")

    def hide_toolbar(self) -> None:
        """Hide the ranges toolbar."""
        self.console.close_nav_drawer()

    def open_explorer(self) -> None:
        """Open the Range Explorer page (shows all ranges)."""
        self.console.command_palette("Open Range Explorer")
        self.page.get_by_text("All Ranges").wait_for(state="visible", timeout=5000)

    def get_toolbar_item(self, name: str) -> Locator:
        """Get a range item locator from the toolbar by name."""
        return self.page.locator(self.TOOLBAR_ITEM_SELECTOR).filter(has_text=name).first

    def get_explorer_item(self, name: str) -> Locator:
        """Get a range item locator from the explorer by name."""
        return (
            self.page.locator(self.EXPLORER_ITEM_SELECTOR).filter(has_text=name).first
        )

    def exists_in_toolbar(self, name: str) -> bool:
        """Check if a range exists in the toolbar (is favorited)."""
        self.show_toolbar()
        items = self.page.locator(self.TOOLBAR_ITEM_SELECTOR).filter(has_text=name)
        return items.count() > 0

    def exists_in_explorer(self, name: str) -> bool:
        """Check if a range exists in the explorer."""
        items = self.page.locator(self.EXPLORER_ITEM_SELECTOR).filter(has_text=name)
        return items.count() > 0

    def create(
        self,
        name: str,
        *,
        persisted: bool = True,
        parent: str | None = None,
        labels: list[str] | None = None,
        stage: str | None = None,
    ) -> None:
        """Create a new range.

        Args:
            name: The name for the new range.
            persisted: If True, saves to Synnax server. If False, saves locally only.
            parent: Optional parent range name to set.
            labels: Optional list of label names to add.
            stage: Optional stage to set ("To Do", "In Progress", "Completed").
        """
        self.console.command_palette("Create a Range")

        modal = self.page.locator(self.CREATE_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)

        name_input = self.page.locator(
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
            stage_button.click()
            self.page.locator(".pluto-list__item").filter(has_text=stage).click(
                timeout=2000
            )

        if parent is not None:
            parent_button = modal.locator("button").filter(has_text="Select a range")
            parent_button.click()
            search_input = self.page.locator("input[placeholder='Search ranges...']")
            search_input.fill(parent)
            self.page.locator(".pluto-range__list-item").filter(has_text=parent).click(
                timeout=5000
            )

        if labels is not None:
            label_button = self.page.get_by_text("Select labels", exact=True)
            label_button.click(timeout=5000)
            for label_name in labels:
                self.page.locator(".pluto-list__item").filter(
                    has_text=label_name
                ).first.click(timeout=2000)
            self.page.keyboard.press("Escape")

        if persisted:
            save_button = self.page.get_by_role("button", name="Save to Synnax")
        else:
            save_button = self.page.get_by_role("button", name="Save Locally")

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
        item = self.get_explorer_item(old_name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        self.page.get_by_text("Rename", exact=True).click(timeout=5000)
        name_input = self.page.locator("input[placeholder='Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(new_name)
        save_btn = self.page.get_by_role("button", name="Save", exact=True)
        save_btn.click(timeout=5000)
        save_btn.wait_for(state="hidden", timeout=5000)
        self.page.wait_for_load_state("networkidle", timeout=5000)

    def delete_from_explorer(self, name: str) -> None:
        """Delete a range via context menu in the explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        self.page.get_by_text("Delete", exact=True).click(timeout=5000)

        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        delete_btn.wait_for(state="hidden", timeout=5000)
        item.wait_for(state="hidden", timeout=5000)

    def favorite_from_explorer(self, name: str) -> None:
        """Add a range to favorites via context menu in the explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        add_btn = self.page.get_by_text("Add to favorites", exact=True)
        remove_btn = self.page.get_by_text("Remove from favorites", exact=True)
        add_btn.or_(remove_btn).wait_for(state="visible", timeout=2000)
        if remove_btn.is_visible():
            self.page.keyboard.press("Escape")
            return
        add_btn.click(timeout=5000)
        add_btn.wait_for(state="hidden", timeout=2000)

    def unfavorite_from_toolbar(self, name: str) -> None:
        """Remove a range from favorites via context menu in the toolbar."""
        self.show_toolbar()
        item = self.get_toolbar_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        remove_btn = self.page.get_by_text("Remove from favorites", exact=True)
        remove_btn.wait_for(state="visible", timeout=5000)
        remove_btn.click()
        remove_btn.wait_for(state="hidden", timeout=2000)

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
        parent_button = self.page.get_by_role("button").filter(has_text=parent_name)
        parent_button.click(timeout=5000)

    def wait_for_overview(self, name: str, timeout: int = 5000) -> None:
        """Wait for the range overview to show a specific range.

        Args:
            name: The name of the range to wait for.
            timeout: Maximum time to wait in milliseconds.
        """
        name_input = self.page.locator(f"input[placeholder='Name'][value='{name}']")
        name_input.wait_for(state="visible", timeout=timeout)

    def is_overview_showing(self, name: str) -> bool:
        """Check if the range overview is showing a specific range.

        Args:
            name: The name of the range to check for.

        Returns:
            True if the overview shows the range name in the header.
        """
        header = self.page.locator("input[placeholder='Name']").first
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
        modal = self.page.locator(".pluto-datetime-modal")
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

        done_btn = self.page.get_by_role("button", name="Done")
        done_btn.click()
        modal.wait_for(state="hidden", timeout=5000)
        self.page.wait_for_load_state("networkidle", timeout=5000)

    def set_start_time_in_overview(
        self,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Set the start time in the range overview.

        Args:
            year: The year (e.g., 2024).
            month: The month name (e.g., "January").
            day: The day of the month (1-31).
            hour: The hour (0-23).
            minute: The minute (0-59).
            second: The second (0-59).
        """
        time_range = self.page.locator(".console-time-range")
        from_btn = time_range.locator("button").first
        from_btn.wait_for(state="visible", timeout=5000)
        self._fill_datetime_picker(from_btn, year, month, day, hour, minute, second)

    def set_end_time_in_overview(
        self,
        year: int,
        month: str,
        day: int,
        hour: int = 0,
        minute: int = 0,
        second: int = 0,
    ) -> None:
        """Set the end time in the range overview.

        Args:
            year: The year (e.g., 2024).
            month: The month name (e.g., "January").
            day: The day of the month (1-31).
            hour: The hour (0-23).
            minute: The minute (0-59).
            second: The second (0-59).
        """
        time_range = self.page.locator(".console-time-range")
        to_btn = time_range.locator("button").nth(1)
        to_btn.wait_for(state="visible", timeout=5000)
        self._fill_datetime_picker(to_btn, year, month, day, hour, minute, second)

    def set_stage_in_overview(self, stage: str) -> None:
        """Set the stage in the range overview.

        Args:
            stage: The stage to set ("To Do", "In Progress", "Completed").
        """
        stage_button = (
            self.page.locator("button")
            .filter(has_text="To Do")
            .or_(self.page.locator("button").filter(has_text="In Progress"))
            .or_(self.page.locator("button").filter(has_text="Completed"))
            .first
        )
        stage_button.click()
        dropdown = self.page.locator(".pluto-list__item").filter(has_text=stage)
        dropdown.click(timeout=2000)
        dropdown.wait_for(state="hidden", timeout=2000)

    def add_label_in_overview(self, label_name: str) -> None:
        """Add a label to the range in the overview.

        Args:
            label_name: The name of the label to add.
        """
        labels_row = self.page.get_by_text("Labels", exact=True).locator("..")
        add_button = labels_row.locator("button").last
        add_button.click()
        item = self.page.locator(".pluto-list__item").filter(has_text=label_name).first
        item.click(timeout=2000)
        self.page.keyboard.press("Escape")
        item.wait_for(state="hidden", timeout=2000)

    def remove_label_in_overview(self, label_name: str) -> None:
        """Remove a label from the range in the overview.

        Args:
            label_name: The name of the label to remove.
        """
        labels_row = self.page.get_by_text("Labels", exact=True).locator("..")
        add_button = labels_row.locator("button").last
        add_button.click()
        item = self.page.locator(".pluto-list__item").filter(has_text=label_name).first
        item.click(timeout=2000)
        self.page.keyboard.press("Escape")
        item.wait_for(state="hidden", timeout=2000)

    def get_labels_in_overview(self) -> list[str]:
        """Get the labels currently attached to the range in the overview.

        Returns:
            A list of label names.
        """
        labels_row = self.page.get_by_text("Labels", exact=True).locator("..")
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
        name_input = self.page.locator("input[placeholder='Name']").first
        name_input.wait_for(state="visible", timeout=5000)
        name_input.click()
        name_input.fill(new_name)
        name_input.blur()
        self.page.wait_for_load_state("networkidle", timeout=5000)

    def copy_python_code_from_overview(self) -> None:
        """Click the Python code copy button in the range overview."""
        python_btn = self.page.locator("button:has(svg.pluto-icon--python)")
        python_btn.click(timeout=5000)

    def copy_typescript_code_from_overview(self) -> None:
        """Click the TypeScript code copy button in the range overview."""
        ts_btn = self.page.locator("button:has(svg.pluto-icon--typescript)")
        ts_btn.click(timeout=5000)

    def copy_link_from_overview(self) -> None:
        """Click the copy link button in the range overview."""
        link_btn = self.page.locator("button:has(svg.pluto-icon--link)")
        link_btn.click(timeout=5000)

    def open_csv_download_modal(self) -> None:
        """Click the CSV download button in the range overview and wait for modal."""
        csv_btn = self.page.locator("button:has(svg.pluto-icon--csv)")
        csv_btn.click(timeout=5000)
        self.page.get_by_text("Download data for").wait_for(
            state="visible", timeout=5000
        )

    def close_csv_download_modal(self) -> None:
        """Close the CSV download modal."""
        close_btn = self.page.locator("button:has(svg.pluto-icon--close)").first
        close_btn.click(timeout=2000)
        self.page.get_by_text("Download data for").wait_for(
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
        self.console.notifications.close_all()
        self.open_csv_download_modal()

        channels_dropdown = self.page.get_by_text("Select channels to download")
        channels_dropdown.click(timeout=5000)
        search_input = self.page.locator("input[placeholder*='Search']")
        search_input.fill(channel)
        self.console.select_from_dropdown(channel)
        self.console.ESCAPE

        download_button = self.page.get_by_role("button", name="Download").last
        self.page.evaluate("delete window.showSaveFilePicker")

        with self.page.expect_download() as download_info:
            download_button.click()

        download = download_info.value
        save_path = get_results_path(f"{range_name}.csv")
        download.save_as(save_path)
        with open(save_path, "r") as f:
            return f.read()

    def _get_child_ranges_section(self) -> Locator:
        """Get the Child Ranges section in the overview."""
        return (
            self.page.get_by_text("Child Ranges", exact=True)
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
        section = self._get_child_ranges_section()
        add_btn = section.locator("button:has(svg.pluto-icon--add)")
        add_btn.click(timeout=5000)
        modal = self.page.locator(self.CREATE_MODAL_SELECTOR)
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
            .filter(has=self.page.locator("svg.pluto-icon--to-do"))
            .or_(
                item.locator("button").filter(
                    has=self.page.locator("svg.pluto-icon--in-progress")
                )
            )
            .or_(
                item.locator("button").filter(
                    has=self.page.locator("svg.pluto-icon--completed")
                )
            )
            .first
        )
        stage_button.click()
        dropdown = self.page.locator(".pluto-list__item").filter(has_text=stage)
        dropdown.click(timeout=2000)
        dropdown.wait_for(state="hidden", timeout=2000)

    def favorite_child_range(self, name: str) -> None:
        """Favorite a child range from the Child Ranges section via context menu.

        Args:
            name: The name of the child range to favorite.
        """
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        add_btn = self.page.get_by_text("Add to favorites", exact=True)
        remove_btn = self.page.get_by_text("Remove from favorites", exact=True)
        add_btn.or_(remove_btn).wait_for(state="visible", timeout=2000)
        if remove_btn.is_visible():
            remove_btn.click()
            remove_btn.wait_for(state="hidden", timeout=2000)
            item.click(button="right")
            add_btn.wait_for(state="visible", timeout=2000)
        add_btn.click()
        add_btn.wait_for(state="hidden", timeout=2000)

    def unfavorite_child_range(self, name: str) -> None:
        """Unfavorite a child range from the Child Ranges section via context menu.

        Args:
            name: The name of the child range to unfavorite.
        """
        item = self.get_child_range_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        remove_btn = self.page.get_by_text("Remove from favorites", exact=True)
        remove_btn.wait_for(state="visible", timeout=2000)
        remove_btn.click()
        remove_btn.wait_for(state="hidden", timeout=2000)

    def child_range_exists(self, name: str) -> bool:
        """Check if a child range exists in the Child Ranges section.

        Args:
            name: The name of the child range.

        Returns:
            True if the child range exists.
        """
        section = self._get_child_ranges_section()
        items = section.locator(".console-range__list-item").filter(has_text=name)
        return items.count() > 0

    def get_label_in_toolbar(self, range_name: str, label_name: str) -> Locator:
        """Get a label tag within a range item in the toolbar."""
        range_item = self.get_toolbar_item(range_name)
        return range_item.locator(".pluto-tag").filter(has_text=label_name).first

    def label_exists_in_toolbar(self, range_name: str, label_name: str) -> bool:
        """Check if a label exists on a range in the toolbar."""
        self.show_toolbar()
        return self.get_label_in_toolbar(range_name, label_name).count() > 0

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
