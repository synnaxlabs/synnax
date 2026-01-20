#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import ABC, abstractmethod
from typing import TYPE_CHECKING

from playwright.sync_api import Locator, Page

from .context_menu import ContextMenu

if TYPE_CHECKING:
    from .console import Console


class ResourceToolbar(ABC):
    """Base class for resource sidebars (channels, ranges, etc.)."""

    # Subclasses must define these
    resource_icon: str  # e.g., "pluto-icon--channel"
    resource_pane_text: str  # e.g., "Channels"
    item_selector: str  # e.g., "div[id^='channel:']"

    def __init__(self, page: Page, console: "Console"):
        """Initialize the resource toolbar.

        Args:
            page: Playwright Page instance
            console: Console instance for UI interactions
        """
        self.page = page
        self.console = console

    @property
    def toolbar_button(self) -> Locator:
        """Get the sidebar toggle button."""
        return self.page.locator("button.console-main-nav__item").filter(
            has=self.page.locator(f"svg.{self.resource_icon}")
        )

    @property
    def pane(self) -> Locator:
        """Get the resource pane."""
        return self.page.locator(f"text={self.resource_pane_text}").first

    @property
    def items(self) -> Locator:
        """Get all items in the resource list."""
        return self.page.locator(self.item_selector)

    def is_visible(self) -> bool:
        """Check if the resource pane is visible."""
        return self.pane.is_visible()

    def show(self) -> None:
        """Show the resource sidebar if not already visible."""
        if not self.is_visible():
            self.toolbar_button.click(force=True, timeout=2000)
            self.pane.wait_for(state="visible", timeout=500)
            self.page.wait_for_timeout(100)

    def hide(self) -> None:
        """Hide the resource sidebar if visible."""
        if self.is_visible():
            self.toolbar_button.click(force=True, timeout=2000)
            self.page.wait_for_timeout(100)

    @abstractmethod
    def find_item(self, name: str) -> Locator:
        """Find a resource item by name.

        Args:
            name: The name of the resource to find

        Returns:
            Locator for the found item
        """
        pass

    def context_menu(self, name: str) -> ContextMenu:
        """Open context menu on a resource item.

        Args:
            name: The name of the resource to open context menu on

        Returns:
            ContextMenu instance for the opened menu
        """
        self.show()
        item = self.find_item(name)
        menu = ContextMenu(self.page)
        menu.open_on(item)
        return menu
