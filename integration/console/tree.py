#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Tree navigation utilities for Console UI automation."""

import time
from typing import TYPE_CHECKING

from playwright.sync_api import Locator, Page
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.context_menu import ContextMenu

if TYPE_CHECKING:
    from console.layout import LayoutClient

_CONTAINER = ".lyra-tree"
_VIRTUALIZER = ".lyra-list__virtualizer"
_SCROLL_STEP = 200
# Rows for a new offset mount on a later frame, and a page the tree has not fetched
# yet needs a round-trip.
_SCROLL_SETTLE_MS = 150
_POLL_MS = 100
_DEFAULT_TIMEOUT_MS = 5000
_SCROLL_TO_TOP_JS = """(el) => {
    const tree = el.closest(".lyra-tree");
    tree.scrollTop += el.getBoundingClientRect().top - tree.getBoundingClientRect().top;
}"""
# Items mount in any order, so the script sorts them by their position in the window.
# The read is complete once it reaches a row outside the subtree or the bottom of the
# tree.
_NESTED_NAMES_JS = """(parent, prefix) => {
    const tree = parent.closest(".lyra-tree");
    const items = [...tree.querySelectorAll("[role=treeitem]")]
        .filter((el) => el.checkVisibility())
        .sort((a, b) => a.offsetTop - b.offsetTop);
    const start = items.indexOf(parent);
    if (start === -1) throw new Error(`tree item ${parent.id} is not visible`);
    const level = Number(parent.getAttribute("aria-level"));
    const names = [];
    for (const el of items.slice(start + 1)) {
        if (Number(el.getAttribute("aria-level")) <= level) return [names, true];
        if (el.id.startsWith(prefix)) names.push(el.innerText.trim());
    }
    const virtualizer = tree.querySelector(".lyra-list__virtualizer");
    if (virtualizer == null) return [names, true];
    const last = items[items.length - 1];
    return [names, last.offsetTop + last.offsetHeight >= virtualizer.scrollHeight];
}"""


class Tree:
    """Utility class for working with ontology tree elements in Console UI.

    Tree items in the Console have IDs in the format `type:uuid`, e.g.:
    - `role:6edfe3a6-898a-4342-b838-a7e6ed292a62`
    - `user:abc123`
    - `channel:xyz789`
    """

    page: Page
    layout: "LayoutClient"
    ctx_menu: ContextMenu

    def __init__(self, layout: "LayoutClient"):
        self.layout = layout
        self.page = layout.page
        self.ctx_menu = layout.ctx_menu

    def by_id(self, item_id: str) -> Locator:
        """Locate a tree item by its ID.

        A windowed tree renumbers its rows whenever the window moves, so an index-bound
        locator points at another item after any scroll. An ID-bound one does not. A
        resource grouped under several expanded parents (a user under each of its
        roles) mounts once per parent with the same ID; any mount is the same
        resource, so this resolves to the first.

        :param item_id: The item ID (e.g., 'channel:42').
        :returns: The Locator for the item.
        """
        return self.page.locator(f"div[id='{item_id}']").first

    def _mounted(self, prefix: str) -> list[tuple[str, str]]:
        """The mounted rows for a prefix, as (ID, text) pairs, deduplicated by ID."""
        rows: list[list[str]] = self.page.locator(f"div[id^='{prefix}']").evaluate_all(
            "(els) => els.filter((el) => el.checkVisibility())"
            ".map((el) => [el.id, el.innerText.trim()])"
        )
        seen: set[str] = set()
        deduped: list[tuple[str, str]] = []
        for item_id, text in rows:
            if item_id in seen:
                continue
            seen.add(item_id)
            deduped.append((item_id, text))
        return deduped

    def find_by_prefix(self, prefix: str) -> list[Locator]:
        """Find all visible tree items with the given ID prefix.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :returns: List of visible Locator elements.
        """
        return [self.by_id(item_id) for item_id, _ in self._mounted(prefix)]

    def _match(self, prefix: str, name: str, *, exact: bool) -> Locator | None:
        for item_id, text in self._mounted(prefix):
            matches = text == name if exact else name.lower() in text.lower()
            if matches:
                return self.by_id(item_id)
        return None

    def _scroll_to(self, offset: int) -> None:
        self.page.locator(_CONTAINER).first.evaluate(
            "(el, top) => { el.scrollTop = top; }", offset
        )
        self.page.wait_for_timeout(_SCROLL_SETTLE_MS)

    def _scroll_step(self) -> int:
        """Scroll distance that keeps every row inside some window of the sweep.

        The window mounts a viewport of rows plus an overscan on each side, so a step
        of one viewport never skips a row.
        """
        height = self.page.locator(_CONTAINER).first.evaluate("(el) => el.clientHeight")
        return max(_SCROLL_STEP, int(height))

    def find_by_name(
        self, prefix: str, name: str, *, exact: bool = True
    ) -> Locator | None:
        """Find a tree item by its ID prefix and display name.

        The tree is windowed, so an item reaches the DOM only once scrolled to. Scrolls
        the tree from the top until the item mounts.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :param name: The display name of the item.
        :param exact: If True, match full text exactly. If False, use substring match.
        :returns: The Locator if found, None otherwise.
        """
        found = self._match(prefix, name, exact=exact)
        if found is not None:
            return found
        virtualizer = self.page.locator(f"{_CONTAINER} {_VIRTUALIZER}").first
        if virtualizer.count() == 0:
            return None
        total = virtualizer.evaluate("(el) => el.scrollHeight")
        for offset in range(0, int(total), self._scroll_step()):
            self._scroll_to(offset)
            found = self._match(prefix, name, exact=exact)
            if found is not None:
                return found
        self._scroll_to(0)
        return None

    def wait_for_name(
        self,
        prefix: str,
        name: str,
        *,
        exact: bool = True,
        timeout: int = _DEFAULT_TIMEOUT_MS,
    ) -> Locator:
        """Wait for a tree item to reach the DOM, sweeping the window on each attempt.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :param name: The display name of the item.
        :param exact: If True, match full text exactly. If False, use substring match.
        :param timeout: Maximum time to wait, in milliseconds.
        :returns: The Locator for the item.
        :raises PlaywrightTimeoutError: If the item never mounts.
        """
        deadline = time.monotonic() + timeout / 1000
        while True:
            found = self.find_by_name(prefix, name, exact=exact)
            if found is not None:
                return found
            if time.monotonic() >= deadline:
                raise PlaywrightTimeoutError(
                    f"Tree item {prefix}{name} did not appear within {timeout}ms"
                )
            self.page.wait_for_timeout(_POLL_MS)

    def wait_for_removal(
        self,
        prefix: str,
        name: str,
        *,
        exact: bool = True,
        timeout: int = _DEFAULT_TIMEOUT_MS,
    ) -> None:
        """Wait for a tree item to leave the tree.

        A windowed tree drops the rows away from the viewport, so absence from the DOM
        proves nothing on its own. Each attempt sweeps the whole tree.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :param name: The display name of the item.
        :param exact: If True, match full text exactly. If False, use substring match.
        :param timeout: Maximum time to wait, in milliseconds.
        :raises PlaywrightTimeoutError: If the item stays in the tree.
        """
        deadline = time.monotonic() + timeout / 1000
        while True:
            if self.find_by_name(prefix, name, exact=exact) is None:
                return
            if time.monotonic() >= deadline:
                raise PlaywrightTimeoutError(
                    f"Tree item {prefix}{name} was still present after {timeout}ms"
                )
            self.page.wait_for_timeout(_POLL_MS)

    def list_names(self, prefix: str) -> list[str]:
        """List the names of the mounted items with the given ID prefix.

        Reads the tree once, so an item that another test removes cannot stall it.

        :param prefix: The ID prefix (e.g., 'role:', 'user:', 'channel:').
        :returns: List of item names.
        """
        return [text for _, text in self._mounted(prefix) if text]

    def get_text(self, item: Locator) -> str:
        """Get the text content from a tree item.

        Looks for text in the editable <p> element first, then falls back to
        inner_text on the item itself.

        :param item: The Locator for the tree item.
        :returns: The text content.
        """
        # Try editable text element first (common in ontology tree items)
        text_element = item.locator("p.lyra-text--editable")
        if text_element.count() > 0:
            return text_element.inner_text().strip()
        return item.inner_text().strip()

    def expand_root(self, prefix: str) -> None:
        """Expand the first root-level tree node with the given prefix.

        Waits for the item to appear, then clicks to expand if not already expanded.

        :param prefix: The ID prefix (e.g., 'project:', 'channel:').
        """
        self.page.locator(f"div[id^='{prefix}']").first.wait_for(
            state="visible", timeout=5000
        )
        items = self.find_by_prefix(prefix)
        if not items:
            return
        root = items[0]
        caret = root.locator(".lyra--location-bottom")
        try:
            caret.wait_for(state="visible", timeout=500)
            return
        except PlaywrightTimeoutError:
            pass
        root.click()
        caret.wait_for(state="visible", timeout=5000)

    def expand_named(self, prefix: str, name: str) -> None:
        """Expand the root-level tree node with the given prefix and display name.

        Unlike expand_root, which expands whichever node sorts first, this
        targets a specific node by name. When several root nodes share a prefix
        (e.g. one project per concurrently-running test), expand_root would
        expand an arbitrary sibling; this expands the intended one.

        :param prefix: The ID prefix (e.g., 'project:').
        :param name: The exact display name of the node to expand.
        """
        self.page.locator(f"div[id^='{prefix}']").first.wait_for(
            state="visible", timeout=5000
        )
        # The tree is windowed, so the node must be swept into the DOM before
        # it can be expanded.
        item = self.wait_for_name(prefix, name)
        item.scroll_into_view_if_needed()
        # An expanded node's caret carries lyra--location-bottom (caret down);
        # collapsed is lyra--location-right. The drawer remounts the tree each
        # time it is reopened, so an expand click issued before the freshly
        # mounted tree settles is silently dropped; retry until the caret flips.
        caret = item.locator(".lyra--location-bottom")
        for _ in range(4):
            try:
                caret.wait_for(state="visible", timeout=500)
                return
            except PlaywrightTimeoutError:
                pass
            item.click()
            try:
                caret.wait_for(state="visible", timeout=3000)
                return
            except PlaywrightTimeoutError:
                continue
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
        expand_icon = item.locator("svg.lyra-tree__expansion-indicator").first
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
        expand_icon = item.locator("svg.lyra-tree__expansion-indicator").first
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
        expand_icon = item.locator("svg.lyra-tree__expansion-indicator").first
        if expand_icon.count() == 0:
            return False
        class_attr = expand_icon.get_attribute("class") or ""
        return "lyra-tree__expansion-indicator--expanded" in class_attr

    def rename(self, item: Locator, new_name: str) -> None:
        """Rename a tree item via context menu.

        :param item: The Locator for the tree item to rename.
        :param new_name: The new name for the item.
        """
        self.ctx_menu.action(item, "Rename")
        self.page.locator(
            "p.lyra-text--editable[contenteditable='true']"
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
        new_group = self.page.locator("div.lyra--selected[id^='group:']").first
        new_group.wait_for(state="visible", timeout=5000)
        new_group.locator("p.lyra-text--editable").dblclick()
        self.layout.press_key("ControlOrMeta+a")
        self.layout.type_text(group_name)
        self.layout.press_enter()

    def delete_group(self, item: Locator | str, *, only_if_empty: bool = False) -> None:
        """Delete/ungroup a group via context menu.

        Accepts either a Locator or a group name string. Groups are deleted
        immediately without a confirmation dialog. The context menu shows
        "Delete" for empty groups and "Ungroup" for non-empty groups.

        :param item: The Locator for the group, or the group name as a string.
        :param only_if_empty: If True, only delete the group when the context
            menu shows "Delete" (empty group). Skips groups that show "Ungroup"
            (still have children). The caller should leave the group expanded so
            the context menu accurately reflects emptiness.
        """
        if isinstance(item, str):
            item = self.get_group(item)
        if not only_if_empty:
            self.collapse(item)
        self.ctx_menu.open_on(item)
        if self.ctx_menu.has_option("Delete"):
            self.ctx_menu.click_option("Delete")
        elif not only_if_empty and self.ctx_menu.has_option("Ungroup"):
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
        self.layout.drag(source, target)
        self.page.wait_for_timeout(300)

    def get_group(self, name: str) -> Locator:
        """Get a group Locator by name, waiting for it to appear and scrolling into
        view.

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

    def has_expand_arrow(self, item: Locator) -> bool:
        """Check if a tree item has an expand arrow (i.e. can have children).

        :param item: The Locator for the tree item.
        :returns: True if the item has an expansion indicator.
        """
        indicator = item.locator("svg.lyra-tree__expansion-indicator")
        return indicator.count() > 0

    def list_nested_names(self, parent: Locator, *, prefix: str) -> list[str]:
        """List the names of the items nested under ``parent``.

        Scrolls ``parent`` to the top of the tree, then reads the tree once, so an item
        that another test removes cannot stall the read.

        :param parent: The Locator for the parent tree item.
        :param prefix: The ID prefix of the nested items (e.g. 'device:').
        :returns: List of item names.
        :raises ValueError: If the nested items do not fit in the tree's window.
        """
        parent.evaluate(_SCROLL_TO_TOP_JS)
        self.page.wait_for_timeout(_SCROLL_SETTLE_MS)
        names: list[str]
        complete: bool
        names, complete = parent.evaluate(_NESTED_NAMES_JS, prefix)
        if not complete:
            raise ValueError(
                f"Items under {parent.get_attribute('id')} do not fit in the tree's "
                "window"
            )
        return names

    def set_editable_text(self, item: Locator, text: str) -> None:
        """Set the editable text content of a tree item.

        Assumes the item is already in edit mode (e.g., after clicking Rename).

        :param item: The Locator for the tree item.
        :param text: The new text to set.
        """
        text_element = item.locator("p.lyra-text--editable")
        text_element.click()
        text_element.fill(text)
        self.layout.press_enter()
