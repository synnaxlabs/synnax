#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING

import synnax as sy
from playwright.sync_api import Locator, Page

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
        toolbar_header.wait_for(state="visible", timeout=5000)

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
        return self.page.locator(self.EXPLORER_ITEM_SELECTOR).filter(has_text=name).first

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
            stage_button = modal.locator("button").filter(has_text="To Do").or_(
                modal.locator("button").filter(has_text="In Progress")
            ).or_(modal.locator("button").filter(has_text="Completed")).first
            stage_button.click()
            self.page.locator(".pluto-list__item").filter(has_text=stage).click(timeout=2000)

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
        self.page.get_by_role("button", name="Save", exact=True).click(timeout=5000)
        sy.sleep(0.1)

    def delete_from_explorer(self, name: str) -> None:
        """Delete a range via context menu in the explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        self.page.get_by_text("Delete", exact=True).click(timeout=5000)

        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        sy.sleep(0.1)

    def favorite_from_explorer(self, name: str) -> None:
        """Add a range to favorites via context menu in the explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        add_btn = self.page.get_by_text("Add to favorites", exact=True)
        remove_btn = self.page.get_by_text("Remove from favorites", exact=True)
        sy.sleep(0.2)
        if remove_btn.is_visible():
            self.page.keyboard.press("Escape")
            return
        add_btn.click(timeout=5000)
        sy.sleep(0.1)

    def unfavorite_from_toolbar(self, name: str) -> None:
        """Remove a range from favorites via context menu in the toolbar."""
        self.show_toolbar()
        item = self.get_toolbar_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        self.page.get_by_text("Remove from favorites", exact=True).click(timeout=5000)
        sy.sleep(0.1)

    def open_overview_from_explorer(self, name: str) -> None:
        """Open the range overview/details page from explorer."""
        item = self.get_explorer_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()

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
        header = self.page.locator("input[placeholder='Name']").first
        header.wait_for(state="visible", timeout=timeout)
        self.page.wait_for_function(
            f"document.querySelector(\"input[placeholder='Name']\").value === '{name}'",
            timeout=timeout,
        )

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
