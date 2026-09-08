#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Context menu helper for Console UI automation."""

import time

from playwright.sync_api import Locator, Page
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

# Menu items may carry a keyboard-trigger badge whose letters join the label in
# textContent ("Rename" renders as "RenameE"), so label matching must strip it.
_ITEM_LABELS_JS = """(els) => els.map((el) => {
    const clone = el.cloneNode(true);
    clone
        .querySelectorAll(".pluto-trigger-indicator")
        .forEach((badge) => badge.remove());
    return (clone.textContent ?? "").trim();
})"""


class ContextMenu:
    """Context menu helper for right-click operations.

    Provides patterns for opening context menus and clicking options,
    searching the entire page for menu options (not scoped to a menu element).
    """

    def __init__(self, page: Page):
        """Initialize the context menu helper.

        Args:
            page: Playwright Page instance.
        """
        self.page = page

    def open_on(self, element: Locator) -> "ContextMenu":
        """Right-click to open context menu.

        Args:
            element: The Playwright Locator to right-click on.

        Returns:
            Self for method chaining.
        """
        element.click(button="right")
        menu = self.page.get_by_role("menu").first
        menu.wait_for(state="visible", timeout=5000)
        menu.get_by_role("menuitem").first.wait_for(state="visible", timeout=2000)
        return self

    def _visible_menu(self) -> Locator:
        """Return the first visible context menu locator."""
        return self.page.get_by_role("menu").first

    def _find_option(self, text: str, *, exact: bool) -> Locator | None:
        """Resolve a menu item by its label, ignoring any trigger-key badge.

        Args:
            text: The label of the menu option to find.
            exact: Whether to match the label exactly or as a substring.

        Returns:
            Locator for the matching menu item, or None when absent.
        """
        items = self._visible_menu().get_by_role("menuitem")
        labels: list[str] = items.evaluate_all(_ITEM_LABELS_JS)
        for index, label in enumerate(labels):
            matches = (label == text) if exact else (text in label)
            if matches:
                return items.nth(index)
        return None

    def click_option(self, text: str, *, exact: bool = True) -> None:
        """Click a menu option by searching within the context menu.

        Waits up to 5s for the option to appear, then waits for the context
        menu to be hidden after clicking.

        Args:
            text: The text of the menu option to click.
            exact: Whether to match text exactly.
        """
        menu = self._visible_menu()
        deadline = time.monotonic() + 5
        option = self._find_option(text, exact=exact)
        while option is None:
            if time.monotonic() >= deadline:
                raise AssertionError(f"context menu offers no option {text!r}")
            self.page.wait_for_timeout(100)
            option = self._find_option(text, exact=exact)
        # Fixed-position menus may extend beyond the viewport, causing
        # both click() and click(force=True) to fail. dispatch_event
        # fires the click via the DOM and does not require the element
        # to be within the viewport.
        option.dispatch_event("click")
        try:
            menu.wait_for(state="hidden", timeout=3000)
        except PlaywrightTimeoutError:
            pass

    def action(self, element: Locator, action_text: str, *, exact: bool = True) -> None:
        """Right-click element and click action in one call.

        Args:
            element: The Playwright Locator to right-click on.
            action_text: The text of the menu action to click.
            exact: Whether to match text exactly.
        """
        self.open_on(element)
        self.click_option(action_text, exact=exact)

    def has_option(self, text: str, *, exact: bool = True) -> bool:
        """Check if a menu option is visible and not disabled.

        Args:
            text: The text of the menu option to check.
            exact: Whether to match text exactly.

        Returns:
            True if the option is visible and not disabled.
        """
        option = self._find_option(text, exact=exact)
        if option is None or not option.is_visible():
            return False
        option_class = option.get_attribute("class") or ""
        return "disabled" not in option_class.lower()

    def close(self) -> None:
        """Close the context menu by clicking outside it.

        Pluto's ContextMenu uses useClickOutside for dismissal.
        """
        menu = self._visible_menu()
        self.page.locator("body").click(position={"x": 1, "y": 1})
        try:
            menu.wait_for(state="hidden", timeout=3000)
        except PlaywrightTimeoutError:
            self.page.keyboard.press("Escape")
