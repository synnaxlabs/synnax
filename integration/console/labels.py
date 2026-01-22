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

if TYPE_CHECKING:
    from .console import Console


_MODAL_SELECTOR = ".console-label__edit"
_LABEL_ITEM_SELECTOR = ".console-label__list-item"


class LabelClient:
    """Console label client for managing labels via the UI."""

    def __init__(self, page: Page, console: "Console"):
        """Initialize the label client.

        Args:
            page: Playwright Page instance
            console: Console instance for UI interactions
        """
        self.page = page
        self.console = console

    def open_edit_modal(self) -> None:
        """Open the Edit Labels modal via command palette.

        Raises:
            RuntimeError: If the modal fails to open.
        """
        self.console.command_palette("Edit Labels")
        modal = self.page.locator(_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        self.page.wait_for_timeout(200)

    def close_edit_modal(self) -> None:
        """Close the Edit Labels modal by clicking the close button."""
        close_button = self.page.locator(
            ".pluto-dialog__dialog button:has(svg.pluto-icon--close)"
        ).first
        close_button.click(timeout=2000)
        modal = self.page.locator(_MODAL_SELECTOR)
        modal.wait_for(state="hidden", timeout=3000)

    def create(self, name: str, color: str | None = None) -> None:
        """Create a new label.

        Args:
            name: The name for the new label.
            color: Optional hex color code (e.g., "#FF0000"). If not provided,
                   uses the default color.
        """

        self.open_edit_modal()
        add_button = self.page.locator(".console-label__add-btn")
        add_button.click(timeout=2000)
        self.page.wait_for_timeout(200)

        create_form = self.page.locator(f"{_LABEL_ITEM_SELECTOR}.console--create").first

        # If color is provided, click the color swatch to open picker
        if color:
            color_swatch = create_form.locator(".pluto-color-swatch").first
            color_swatch.click(timeout=1000)
            self.page.wait_for_timeout(100)
            # The color picker should open - find and fill the hex input
            # or select from the preset colors
            # For now, try clicking the swatch again to toggle and use preset
            self.page.keyboard.press("Escape")  # Close color picker

        name_input = create_form.locator("input[placeholder='Label Name']")
        name_input.fill(name)
        self.page.wait_for_timeout(100)

        save_button = create_form.locator("button:has(svg.pluto-icon--check)")
        save_button.click(timeout=2000)
        self.page.wait_for_timeout(300)  # Wait for save to complete
        self.close_edit_modal()

    def list_all(self) -> list[str]:
        """List all existing labels.

        Returns:
            List of label names.
        """
        self.open_edit_modal()

        labels: list[str] = []
        # Get all label items (excluding the create form)
        label_items = self.page.locator(
            f"{_LABEL_ITEM_SELECTOR}:not(.console--create)"
        ).all()

        for item in label_items:
            if item.is_visible():
                # Get the label name from the input field
                name_input = item.locator("input").first
                if name_input.count() > 0:
                    name = name_input.input_value()
                    if name:
                        labels.append(name)

        return labels

    def assert_exists(self, name: str) -> None:
        """Assert a label exists by name. Throws an error if the label is not found.

        Args:
            name: The name of the label to find.
        """
        self.open_edit_modal()
        label_item = self._find_label_item(name)
        if label_item is None:
            raise ValueError(f"Label '{name}' not found")
        self.close_edit_modal()

    def _find_label_item(self, name: str) -> Locator | None:
        """Find a label item by name.

        Args:
            name: The name of the label to find.

        Returns:
            The Locator for the label item, or None if not found.
        """
        label_items = self.page.locator(
            f"{_LABEL_ITEM_SELECTOR}:not(.console--create)"
        ).all()

        for item in label_items:
            if item.is_visible():
                name_input = item.locator("input").first
                if name_input.count() > 0:
                    current_name = name_input.input_value()
                    if current_name == name:
                        return item
        return None

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename an existing label.

        Args:
            old_name: The current name of the label.
            new_name: The new name for the label.
        """
        # Open the modal if not already open
        modal = self.page.locator(_MODAL_SELECTOR)
        if not modal.is_visible():
            self.open_edit_modal()

        label_item = self._find_label_item(old_name)
        if label_item is None:
            raise ValueError(f"Label '{old_name}' not found")

        # Click on the name input and change it
        name_input = label_item.locator("input").first
        name_input.click()
        name_input.clear()
        name_input.type(new_name)
        # Press Tab to blur and trigger auto-save
        self.page.keyboard.press("Tab")
        self.page.wait_for_timeout(500)  # Wait for save to complete
        self.close_edit_modal()

    def delete(self, name: str) -> None:
        """Delete a label.

        Args:
            name: The name of the label to delete.
        """
        # Open the modal if not already open
        modal = self.page.locator(_MODAL_SELECTOR)
        if not modal.is_visible():
            self.open_edit_modal()

        # Find the label item
        label_item = self._find_label_item(name)
        if label_item is None:
            raise ValueError(f"Label '{name}' not found")

        # Hover over the item to show the delete button
        label_item.hover()
        self.page.wait_for_timeout(100)

        # Click the delete button (has pluto-icon--delete)
        delete_button = label_item.locator("button:has(svg.pluto-icon--delete)")
        delete_button.click(timeout=2000)
        self.close_edit_modal()
        self.page.wait_for_timeout(300)  # Wait for delete to complete

    def change_color(self, name: str, new_color: str) -> None:
        """Change the color of a label.

        Args:
            name: The name of the label to update.
            new_color: The new hex color code (e.g., "#FF0000").
        """
        self.open_edit_modal()

        label_item = self._find_label_item(name)
        if label_item is None:
            raise ValueError(f"Label '{name}' not found")

        # Click the color swatch to open the picker
        color_swatch = label_item.locator(".pluto-color-swatch").first
        color_swatch.click(timeout=2000)
        self.page.wait_for_timeout(200)

        # The SketchPicker has a hex input - find and fill it
        # The input is in the color picker container
        hex_input = self.page.locator(".sketch-picker input").first
        hex_input.click()
        hex_input.fill(new_color.lstrip("#"))
        self.page.keyboard.press("Enter")
        self.page.wait_for_timeout(200)

        # Click outside to close the color picker
        label_item.locator("input").first.click()
        self.page.wait_for_timeout(300)
        self.close_edit_modal()
