#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Context menu helper for Console UI automation."""

from playwright.sync_api import Error as PlaywrightError
from playwright.sync_api import Locator, Page
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

MENU_SELECTOR = ".pluto-menu-context"


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
        menu = self.page.locator(MENU_SELECTOR).first
        menu.wait_for(state="visible", timeout=5000)
        menu.locator("[class*='menu-item']").first.wait_for(
            state="visible", timeout=2000
        )
        return self

    def _visible_menu(self) -> Locator:
        """Return the first visible context menu locator."""
        return self.page.locator(f"{MENU_SELECTOR}:visible").first

    def click_option(self, text: str, *, exact: bool = True) -> None:
        """Click a menu option by searching within the context menu.

        After clicking, waits for the context menu to be hidden.

        Args:
            text: The text of the menu option to click.
            exact: Whether to match text exactly.
        """
        menu = self._visible_menu()
        option = menu.get_by_text(text, exact=exact).first
        option.wait_for(state="visible", timeout=5000)
        try:
            option.click(timeout=5000)
        except (PlaywrightTimeoutError, PlaywrightError):
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
        menu = self._visible_menu()
        option = menu.get_by_text(text, exact=exact).first
        cnt = option.count()
        vis = option.is_visible() if cnt > 0 else False
        if cnt == 0 or not vis:
            print(
                f"[has_option] text={text!r} count={cnt} visible={vis} "
                f"menu_visible={menu.is_visible()} "
                f"all_menus={self.page.locator(MENU_SELECTOR).count()}",
                flush=True,
            )
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
