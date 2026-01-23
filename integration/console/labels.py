#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from re import search as re_search
from typing import TYPE_CHECKING

from playwright.sync_api import Locator, Page

from framework.utils import rgb_to_hex

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

    def create(self, name: str, *, color: str | None = None) -> None:
        """Create a new label.

        Args:
            name: The name for the new label.
            color: Optional hex color code (e.g., "#FF0000") to set for the new label.
        """
        self._open_edit_modal()
        add_button = self.page.locator(".console-label__add-btn")
        add_button.click()

        create_form = self.page.locator(f"{_LABEL_ITEM_SELECTOR}.console--create").first

        if color is not None:
            color_swatch = create_form.locator(".pluto-color-swatch").first
            color_swatch.click()
            hex_input = self.page.locator(".sketch-picker input").first
            hex_input.click()
            hex_input.fill(color.lstrip("#"))
            self.page.keyboard.press("Enter")
            self.page.keyboard.press("Escape")

        name_input = create_form.locator("input[placeholder='Label Name']")
        name_input.fill(name)

        save_button = create_form.locator("button:has(svg.pluto-icon--check)")
        save_button.click()
        create_form.wait_for(state="hidden", timeout=5000)

        label_item = self._find_label_item(name)
        if label_item is None:
            items = self._find_label_items()
            all_names = self._enumerate_label_names(items)
            raise ValueError(
                f"Label '{name}' was not created successfully. Available labels: {all_names}"
            )

        self._close_edit_modal()

    def exists(self, name: str) -> bool:
        """Check if a label exists by name."""
        self._open_edit_modal()
        label_item = self._find_label_item(name)
        exists = label_item is not None
        self._close_edit_modal()
        return exists

    def get_color(self, name: str) -> str:
        """Get the color of a label by name."""
        self._open_edit_modal()
        label_item = self._find_label_item(name)
        if label_item is None:
            raise ValueError(f"Label '{name}' not found")
        color_swatch = label_item.locator(".pluto-color-swatch").first
        style = color_swatch.get_attribute("style")
        if style is None:
            raise ValueError(f"Label '{name}' has no style attribute")
        # Find --pluto-swatch-color: rgba( ... );
        match = re_search(r"--pluto-swatch-color:\s*(rgba?\([^)]+\))", style)
        if match is None:
            raise ValueError(f"Label '{name}' does not have --pluto-swatch-color")
        rgba = match.group(1)
        color = rgb_to_hex(rgba)
        self._close_edit_modal()
        return color

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename an existing label.

        Args:
            old_name: The current name of the label.
            new_name: The new name for the label.
        """
        self._open_edit_modal()

        label_item = self._find_label_item(old_name)
        if label_item is None:
            raise ValueError(f"Label '{old_name}' not found")

        name_input = label_item.locator("input[placeholder='Label Name']").first

        name_input.click()
        self.console.select_all()

        self.page.keyboard.type(new_name)

        name_input.press("Enter")

        self.page.wait_for_load_state("networkidle", timeout=5000)

        renamed_item = self._find_label_item(new_name)

        if renamed_item is None:
            all_items = self._find_label_items()
            all_names = self._enumerate_label_names(all_items)
            raise RuntimeError(
                f"Label rename from '{old_name}' to '{new_name}' did not complete. Available labels: {all_names}"
            )

        self._close_edit_modal()

    def delete(self, name: str) -> None:
        """Delete a label.

        Args:
            name: The name of the label to delete.

        Raises:
            ValueError: If the label with the provided name is not found.
        """
        self._open_edit_modal()

        label_item = self._find_label_item(name)
        if label_item is None:
            items = self._find_label_items()
            all_names = self._enumerate_label_names(items)
            self._close_edit_modal()
            raise ValueError(f"Label '{name}' not found. Available labels: {all_names}")

        label_item.hover()

        delete_button = label_item.locator("button.console-label__delete")
        delete_button.wait_for(state="visible", timeout=5000)

        element_id = label_item.get_attribute("id")
        delete_button.click(timeout=5000)

        self.page.locator(f"[id='{element_id}']").wait_for(
            state="hidden", timeout=10000
        )

        still_exists = self._find_label_item(name)
        if still_exists is not None:
            raise RuntimeError(
                f"Failed to delete label '{name}' - still exists after clicking delete"
            )

        self._close_edit_modal()

    def list_all(self) -> list[str]:
        """List all existing labels.

        Returns:
            List of label names.
        """
        self._open_edit_modal()
        labels: list[str] = []
        items = self._find_label_items()
        for item in items:
            if item.is_visible():
                name_input = item.locator("input[placeholder='Label Name']").first
                if name_input.count() > 0:
                    name = name_input.input_value()
                    if name:
                        labels.append(name)
        self._close_edit_modal()
        return labels

    def change_color(self, *, name: str, new_color: str) -> None:
        """Change the color of a label.

        Args:
            name: The name of the label to update.
            new_color: The new hex color code (e.g., "#FF0000").
        """
        self._open_edit_modal()

        label_item = self._find_label_item(name)
        if label_item is None:
            raise ValueError(f"Label '{name}' not found")

        color_swatch = label_item.locator(".pluto-color-swatch").first
        color_swatch.click()

        hex_input = self.page.locator(".sketch-picker input").first
        hex_input.click()
        hex_input.fill(new_color.lstrip("#"))
        self.page.keyboard.press("Enter")
        self.page.keyboard.press("Escape")
        self._close_edit_modal()

    def _open_edit_modal(self) -> None:
        self.console.command_palette("Edit Labels")
        modal = self.page.locator(_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)

    def _close_edit_modal(self) -> None:
        close_button = self.page.locator(
            ".pluto-dialog__dialog button:has(svg.pluto-icon--close)"
        ).first
        close_button.click()
        modal = self.page.locator(_MODAL_SELECTOR)
        modal.wait_for(state="hidden", timeout=5000)

    def _find_label_item(self, name: str) -> Locator | None:
        for item in self._find_label_items():
            if not item.is_visible():
                continue
            name_input = item.locator("input[placeholder='Label Name']").first
            if name_input.count() == 0:
                continue
            if name_input.input_value() != name:
                continue
            element_id = item.get_attribute("id")
            return self.page.locator(f"[id='{element_id}']")
        return None

    def _find_label_items(self) -> list[Locator]:
        return self.page.locator(f"{_LABEL_ITEM_SELECTOR}:not(.console--create)").all()

    def _enumerate_label_names(self, items: list[Locator]) -> list[str]:
        """Extract label names from a list of label item locators.

        Args:
            items: List of label item locators

        Returns:
            List of label names
        """
        all_names = []
        for item in items:
            if item.is_visible():
                inp = item.locator("input[placeholder='Label Name']").first
                if inp.count() > 0:
                    current_name = inp.input_value()
                    if current_name:
                        all_names.append(current_name)
        return all_names
