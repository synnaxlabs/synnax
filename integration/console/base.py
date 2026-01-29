#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Base client classes for Console UI automation.

Provides shared patterns and helpers that all console clients can inherit from.
"""

from typing import TYPE_CHECKING

import synnax as sy
from playwright.sync_api import Locator

if TYPE_CHECKING:
    from .layout import LayoutClient
    from .notifications import NotificationsClient


class BaseClient:
    """Base class for all console clients with shared patterns.

    Provides common methods for context menus, panel navigation, modal handling,
    and other UI patterns used across multiple clients.
    """

    MODAL_SELECTOR = "div.pluto-dialog__dialog.pluto--modal.pluto--visible"

    layout: "LayoutClient"

    def __init__(self, layout: "LayoutClient"):
        """Initialize the base client.

        Args:
            layout: The LayoutClient for UI operations.
        """
        self.layout = layout

    def _right_click(self, item: Locator) -> None:
        """Right-click on an item to open context menu.

        Args:
            item: The Locator for the element to right-click.
        """
        item.click(button="right")
        sy.sleep(0.2)

    def _wait_for_hidden(self, item: Locator, timeout: int = 5000) -> None:
        """Wait for an item to be removed/hidden.

        Args:
            item: The Locator for the element to wait for.
            timeout: Maximum time in milliseconds to wait.
        """
        item.wait_for(state="hidden", timeout=timeout)

    def _context_menu_action(self, item: Locator, action: str) -> None:
        """Perform a context menu action on an item.

        Args:
            item: The Locator for the element to right-click.
            action: The exact text of the menu action to click.
        """
        self._right_click(item)
        self.layout.page.get_by_text(action, exact=True).click()

    def _show_panel_by_icon(self, icon_name: str, item_prefix: str) -> None:
        """Show a navigation panel by clicking its toolbar button.

        Args:
            icon_name: The icon class suffix (e.g., "device", "user", "channel").
            item_prefix: The ID prefix of items in the panel (e.g., "rack:", "role:").
        """
        items = self.layout.page.locator(f"div[id^='{item_prefix}']")
        if items.count() > 0 and items.first.is_visible():
            return

        button = self.layout.page.locator("button.console-main-nav__item").filter(
            has=self.layout.page.locator(f"svg.pluto-icon--{icon_name}")
        )
        button.click(timeout=5000)
        items.first.wait_for(state="visible", timeout=5000)

    def _show_toolbar(self, shortcut_key: str, item_prefix: str) -> None:
        """Show a navigation toolbar using keyboard shortcut.

        Args:
            shortcut_key: The keyboard shortcut (e.g., "d", "u", "r").
            item_prefix: The ID prefix of items in the panel (e.g., "rack:", "role:").
        """
        items = self.layout.page.locator(f"div[id^='{item_prefix}']")
        if items.count() > 0 and items.first.is_visible():
            return
        self.layout.press_key(shortcut_key)
        items.first.wait_for(state="visible", timeout=5000)

    def _find_toolbar_item(self, item_prefix: str, name: str) -> Locator | None:
        """Find a toolbar item by name.

        Args:
            item_prefix: The ID prefix of items (e.g., "rack:", "role:").
            name: The name to search for.

        Returns:
            The Locator for the item, or None if not found.
        """
        items = self.layout.page.locator(f"div[id^='{item_prefix}']").filter(
            has_text=name
        )
        if items.count() == 0:
            return None
        return items.first

    def _toolbar_item_exists(self, item_prefix: str, name: str) -> bool:
        """Check if a toolbar item exists.

        Args:
            item_prefix: The ID prefix of items (e.g., "rack:", "role:").
            name: The name to search for.

        Returns:
            True if the item exists, False otherwise.
        """
        return self._find_toolbar_item(item_prefix, name) is not None

    def _get_toolbar_item(self, item_prefix: str, name: str) -> Locator:
        """Get a toolbar item by name, raising if not found.

        Args:
            item_prefix: The ID prefix of items (e.g., "rack:", "role:").
            name: The name to search for.

        Returns:
            The Locator for the item.

        Raises:
            ValueError: If the item is not found.
        """
        item = self._find_toolbar_item(item_prefix, name)
        if item is None:
            raise ValueError(
                f"Item '{name}' not found in toolbar (prefix: {item_prefix})"
            )
        return item

    def _open_modal(self, command: str, selector: str) -> None:
        """Open a modal via command palette.

        Args:
            command: The command to execute in the palette.
            selector: CSS selector for the modal to wait for.
        """
        self.layout.command_palette(command)
        self.layout.page.locator(selector).wait_for(state="visible", timeout=5000)

    def _close_modal(self, selector: str) -> None:
        """Close a modal via close button.

        Args:
            selector: CSS selector for the modal to wait for hidden.
        """
        close_btn = self.layout.page.locator(
            ".pluto-dialog__dialog button:has(svg.pluto-icon--close)"
        ).first
        close_btn.click()
        self.layout.page.locator(selector).wait_for(state="hidden", timeout=5000)


class BaseClientWithNotifications(BaseClient):
    """Base class for clients that need notification checking.

    Extends BaseClient with notification-related helpers for clients
    that need to check for error notifications.
    """

    notifications: "NotificationsClient"

    def __init__(self, layout: "LayoutClient", notifications: "NotificationsClient"):
        """Initialize the client with layout and notifications.

        Args:
            layout: The LayoutClient for UI operations.
            notifications: The NotificationsClient for checking/closing notifications.
        """
        super().__init__(layout)
        self.notifications = notifications

    def _check_for_errors(self) -> bool:
        """Check notifications for errors.

        Returns:
            True if errors were found, False otherwise.
        """
        for notification in self.notifications.check():
            message = notification.get("message", "")
            if "Failed" in message or "Error" in message:
                self.notifications.close(0)
                return True
        return False
