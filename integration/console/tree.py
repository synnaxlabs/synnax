#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Tree navigation utilities for Console UI automation."""

from playwright.sync_api import Locator, Page

import synnax as sy


class Tree:
    """Utility class for working with ontology tree elements in Console UI.

    Tree items in the Console have IDs in the format `type:uuid`, e.g.:
    - `role:6edfe3a6-898a-4342-b838-a7e6ed292a62`
    - `user:abc123`
    - `channel:xyz789`
    """

    page: Page

    def __init__(self, page: Page):
        self.page = page

    def find_by_prefix(self, prefix: str) -> list[Locator]:
        """Find all visible tree items with the given ID prefix.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :returns: List of visible Locator elements.
        """
        elements = self.page.locator(f"div[id^='{prefix}']").all()
        return [el for el in elements if el.is_visible()]

    def find_by_name(self, prefix: str, name: str) -> Locator | None:
        """Find a tree item by its ID prefix and display name.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :param name: The display name of the item.
        :returns: The Locator if found, None otherwise.
        """
        elements = self.find_by_prefix(prefix)
        for element in elements:
            text = element.inner_text().strip()
            if text == name:
                return element
        return None

    def list_names(self, prefix: str) -> list[str]:
        """List all visible item names with the given ID prefix.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :returns: List of item names.
        """
        elements = self.find_by_prefix(prefix)
        names: list[str] = []
        for element in elements:
            text = self.get_text(element)
            if text:
                names.append(text)
        return names

    def get_text(self, item: Locator) -> str:
        """Get the text content from a tree item.

        Looks for text in the editable <p> element first, then falls back to
        inner_text on the item itself.

        :param item: The Locator for the tree item.
        :returns: The text content.
        """
        # Try editable text element first (common in ontology tree items)
        text_element = item.locator("p.pluto-text--editable")
        if text_element.count() > 0:
            return text_element.inner_text().strip()
        return item.inner_text().strip()

    def expand(self, item: Locator) -> None:
        """Expand a tree node to show its children.

        Uses single click on the expansion indicator (caret icon).
        If no expansion indicator is found, clicks the item directly.

        :param item: The Locator for the tree item to expand.
        """
        expand_icon = item.locator("svg.pluto-tree__expansion-indicator").first
        if expand_icon.count() > 0:
            expand_icon.click()
        else:
            item.click()
        sy.sleep(0.5)

    def collapse(self, item: Locator) -> None:
        """Collapse a tree node to hide its children.

        Same as expand - single click toggles the state.

        :param item: The Locator for the tree item to collapse.
        """
        self.expand(item)

    def is_expanded(self, item: Locator) -> bool:
        """Check if a tree node is currently expanded.

        :param item: The Locator for the tree item to check.
        :returns: True if expanded, False otherwise.
        """
        # Check for expanded class on the expansion indicator
        expand_icon = item.locator("svg.pluto-tree__expansion-indicator").first
        if expand_icon.count() == 0:
            return False
        class_attr = expand_icon.get_attribute("class") or ""
        return "pluto-tree__expansion-indicator--expanded" in class_attr

    def right_click(self, item: Locator) -> None:
        """Right-click on a tree item to open context menu.

        :param item: The Locator for the tree item.
        """
        item.click(button="right")
        sy.sleep(0.2)

    def get_editable_text(self, item: Locator) -> str:
        """Get the editable text content from a tree item.

        :param item: The Locator for the tree item.
        :returns: The text content.
        """
        text_element = item.locator("p.pluto-text--editable")
        if text_element.count() > 0:
            return text_element.inner_text().strip()
        return item.inner_text().strip()

    def set_editable_text(self, item: Locator, text: str) -> None:
        """Set the editable text content of a tree item.

        Assumes the item is already in edit mode (e.g., after clicking Rename).

        :param item: The Locator for the tree item.
        :param text: The new text to set.
        """
        text_element = item.locator("p.pluto-text--editable")
        text_element.click()
        text_element.fill(text)
        self.page.keyboard.press("Enter")
        sy.sleep(0.2)
