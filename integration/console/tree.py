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
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.context_menu import ContextMenu
from console.layout import LayoutClient


class Tree:
    """Utility class for working with ontology tree elements in Console UI.

    Tree items in the Console have IDs in the format `type:uuid`, e.g.:
    - `role:6edfe3a6-898a-4342-b838-a7e6ed292a62`
    - `user:abc123`
    - `channel:xyz789`
    """

    page: Page
    layout: LayoutClient
    ctx_menu: ContextMenu

    def __init__(self, page: Page):
        self.page = page
        self.layout = LayoutClient(page)
        self.ctx_menu = ContextMenu(page)

    def find_by_prefix(self, prefix: str) -> list[Locator]:
        """Find all visible tree items with the given ID prefix.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :returns: List of visible Locator elements.
        """
        locator = self.page.locator(f"div[id^='{prefix}']")
        elements = locator.all()
        return [el for el in elements if el.is_visible()]

    def find_by_name(
        self, prefix: str, name: str, *, exact: bool = True
    ) -> Locator | None:
        """Find a tree item by its ID prefix and display name.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :param name: The display name of the item.
        :param exact: If True, match full text exactly. If False, use substring match.
        :returns: The Locator if found, None otherwise.
        """
        if exact:
            elements = self.find_by_prefix(prefix)
            for element in elements:
                text = element.inner_text().strip()
                if text == name:
                    return element
            return None
        else:
            items = self.page.locator(f"div[id^='{prefix}']").filter(has_text=name)
            if items.count() == 0:
                return None
            return items.first

    def wait_for_removal(self, prefix: str, name: str, *, exact: bool = True) -> None:
        """Wait for a tree item to be removed.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :param name: The display name of the item.
        :param exact: If True, match full text exactly. If False, use substring match.
        """
        if exact:
            item = self.page.locator(f"div[id^='{prefix}']").filter(
                has=self.page.get_by_text(name, exact=True)
            )
        else:
            item = self.page.locator(f"div[id^='{prefix}']").filter(has_text=name)
        item.first.wait_for(state="hidden", timeout=5000)

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

    def expand_root(self, prefix: str) -> None:
        """Expand the first root-level tree node with the given prefix.

        Waits for the item to appear, then clicks to expand if not already expanded.

        :param prefix: The ID prefix (e.g., 'workspace:', 'channel:').
        """
        self.page.locator(f"div[id^='{prefix}']").first.wait_for(
            state="visible", timeout=5000
        )
        items = self.find_by_prefix(prefix)
        if not items:
            return
        root = items[0]
        caret = root.locator(".pluto--location-bottom")
        try:
            caret.wait_for(state="visible", timeout=500)
            return
        except PlaywrightTimeoutError:
            pass
        root.click()
        caret.wait_for(state="visible", timeout=5000)

    def expand(self, item: Locator) -> None:
        """Expand a tree node to show its children.

        Checks if the node is already expanded first to avoid toggling it
        closed. This makes the operation idempotent and safe for concurrent
        callers.

        :param item: The Locator for the tree item to expand.
        """
        if self.is_expanded(item):
            return
        expand_icon = item.locator("svg.pluto-tree__expansion-indicator").first
        if expand_icon.count() > 0:
            expand_icon.click()
        else:
            item.click()
        self.page.wait_for_timeout(300)

    def collapse(self, item: Locator) -> None:
        """Collapse a tree node to hide its children.

        Checks if the node is expanded first to avoid toggling it open.

        :param item: The Locator for the tree item to collapse.
        """
        if not self.is_expanded(item):
            return
        expand_icon = item.locator("svg.pluto-tree__expansion-indicator").first
        if expand_icon.count() > 0:
            expand_icon.click()
        else:
            item.click()
        self.page.wait_for_timeout(300)

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

    def get_editable_text(self, item: Locator) -> str:
        """Get the editable text content from a tree item.

        :param item: The Locator for the tree item.
        :returns: The text content.
        """
        text_element = item.locator("p.pluto-text--editable")
        if text_element.count() > 0:
            return text_element.inner_text().strip()
        return item.inner_text().strip()

    def rename(self, item: Locator, new_name: str) -> None:
        """Rename a tree item via context menu.

        :param item: The Locator for the tree item to rename.
        :param new_name: The new name for the item.
        """
        self.ctx_menu.action(item, "Rename")
        self.page.locator(
            "p.pluto-text--editable[contenteditable='true']"
        ).first.wait_for(state="visible", timeout=5000)
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()

    def rename_group(self, old_name: str, new_name: str) -> None:
        """Rename a group by name via context menu.

        :param old_name: Current name of the group.
        :param new_name: New name for the group.
        """
        item = self.get_group(old_name)
        self.rename(item, new_name)

    def group(self, items: list[Locator], group_name: str) -> None:
        """Group multiple tree items into a new group via multi-select and context menu.

        :param items: List of Locators for the tree items to group.
        :param group_name: Name for the new group.
        """
        if not items:
            return
        items[0].click()
        for item in items[1:]:
            item.click(modifiers=["ControlOrMeta"])
        last_item = items[-1]
        self.ctx_menu.action(last_item, "Group selection")
        # "Group selection" creates a group but doesn't enter name edit mode.
        # The newly created group gets the "selected" class.
        new_group = self.page.locator("div.pluto--selected[id^='group:']").first
        new_group.wait_for(state="visible", timeout=5000)
        new_group.locator("p.pluto-text--editable").dblclick()
        self.layout.press_key("ControlOrMeta+a")
        self.layout.type_text(group_name)
        self.layout.press_enter()

    def delete_group(self, item: Locator | str) -> None:
        """Delete/ungroup a group via context menu.

        Accepts either a Locator or a group name string. Groups are deleted
        immediately without a confirmation dialog. The context menu shows
        "Delete" for collapsed groups and "Ungroup" for expanded groups.

        :param item: The Locator for the group, or the group name as a string.
        """
        if isinstance(item, str):
            item = self.get_group(item)
        self.ctx_menu.open_on(item)
        if self.ctx_menu.has_option("Delete"):
            self.ctx_menu.click_option("Delete")
        elif self.ctx_menu.has_option("Ungroup"):
            self.ctx_menu.click_option("Ungroup")
        else:
            self.ctx_menu.close()

    def move_to_group(self, source: Locator, group_name: str) -> None:
        """Move a tree item into a group via drag-and-drop.

        :param source: Locator for the item to move.
        :param group_name: Name of the target group.
        """
        target = self.find_by_name("group:", group_name)
        if target is None:
            raise ValueError(f"Group '{group_name}' not found")
        source.drag_to(target)
        self.page.wait_for_timeout(300)

    def get_group(self, name: str) -> Locator:
        """Get a group Locator by name, waiting for it to appear and scrolling into view.

        More robust than find_by_name("group:", name) which filters by visibility.
        This waits for the element to be attached to the DOM and scrolls it into view.

        :param name: The display name of the group.
        :returns: The Locator for the group item.
        """
        item = self.page.locator("div[id^='group:']").filter(
            has=self.page.get_by_text(name, exact=True)
        )
        item.first.wait_for(state="attached", timeout=5000)
        item.first.scroll_into_view_if_needed()
        return item.first

    def group_exists(self, name: str) -> bool:
        """Check if a group is visible in the tree.

        :param name: The display name of the group.
        :returns: True if the group is visible, False otherwise.
        """
        try:
            self.get_group(name).wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

    def set_editable_text(self, item: Locator, text: str) -> None:
        """Set the editable text content of a tree item.

        Assumes the item is already in edit mode (e.g., after clicking Rename).

        :param item: The Locator for the tree item.
        :param text: The new text to set.
        """
        text_element = item.locator("p.pluto-text--editable")
        text_element.click()
        text_element.fill(text)
        self.layout.press_enter()
