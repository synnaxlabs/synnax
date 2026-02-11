#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Context menu helper for Console UI automation."""

from playwright.sync_api import Locator, Page


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
        self.page.locator(".pluto-menu-context").first.wait_for(
            state="visible", timeout=5000
        )
        return self

    def click_option(self, text: str, *, exact: bool = True) -> None:
        """Click a menu option by searching within the context menu.

        Searches within the visible context menu for the text.
        Waits for the option to be visible before clicking.

        Args:
            text: The text of the menu option to click.
            exact: Whether to match text exactly.
        """
        menu = self.page.locator(".pluto-menu-context")
        option = menu.get_by_text(text, exact=exact).first
        option.wait_for(state="visible", timeout=2000)
        option.click()

    def action(self, element: Locator, action_text: str, *, exact: bool = True) -> None:
        """Right-click element and click action in one call.

        Args:
            element: The Playwright Locator to right-click on.
            action_text: The text of the menu action to click.
            exact: Whether to match text exactly.
        """
        self.open_on(element)
        self.click_option(action_text, exact=exact)

    def close(self) -> None:
        """Close the context menu by pressing Escape."""
        self.page.keyboard.press("Escape")
