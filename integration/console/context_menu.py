#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator, Page


class ContextMenu:
    """Context menu interaction helper for right-click operations."""

    def __init__(self, page: Page):
        """Initialize the context menu helper.

        Args:
            page: Playwright Page instance
        """
        self.page = page
        self._menu_selector = ".pluto-menu"

    def open_on(self, element: Locator) -> "ContextMenu":
        """Open context menu by right-clicking on an element.

        Args:
            element: The Playwright Locator to right-click on

        Returns:
            Self for method chaining
        """
        element.click(button="right")
        self.page.locator(self._menu_selector).wait_for(state="visible", timeout=2000)
        self.page.wait_for_timeout(100)  # Wait for menu to fully render
        return self

    def click_option(self, text: str) -> None:
        """Click a menu option by text.

        Args:
            text: The text of the menu option to click
        """
        option = self.page.get_by_text(text, exact=True).first
        option.wait_for(state="visible", timeout=1000)
        option.click()
        self.page.wait_for_timeout(100)

    def has_option(self, text: str) -> bool:
        """Check if a menu option exists.

        Args:
            text: The text of the menu option to check

        Returns:
            True if the option exists, False otherwise
        """
        menu = self.page.locator(self._menu_selector)
        if not menu.is_visible():
            return False
        option = menu.get_by_text(text, exact=True)
        return option.count() > 0

    def close(self) -> None:
        """Close the context menu by pressing Escape."""
        self.page.keyboard.press("Escape")
        self.page.wait_for_timeout(100)
