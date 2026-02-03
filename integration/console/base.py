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

from playwright.sync_api import Locator

from .context_menu import ContextMenu
from .layout import LayoutClient
from .notifications import NotificationsClient
from .tree import Tree


class BaseClient:
    """Base class for all console clients with shared patterns.

    Provides common methods for context menus, panel navigation, modal handling,
    and other UI patterns used across multiple clients.
    """

    MODAL_SELECTOR = "div.pluto-dialog__dialog.pluto--modal.pluto--visible"

    layout: LayoutClient

    def __init__(self, layout: LayoutClient):
        """Initialize the base client.

        Args:
            layout: The LayoutClient for UI operations.
        """
        self.layout = layout
        self.ctx_menu = ContextMenu(layout.page)
        self.notifications = NotificationsClient(layout.page)
        self.tree = Tree(layout.page)

    def _context_menu_action(self, item: Locator, action: str) -> None:
        """Perform a context menu action on an item.

        Args:
            item: The Locator for the element to right-click.
            action: The exact text of the menu action to click.
        """
        self.ctx_menu.action(item, action)

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

    def _delete_with_confirmation(self, item: Locator) -> None:
        """Delete an item via context menu with confirmation modal."""
        self.ctx_menu.action(item, "Delete")
        modal = self.layout.page.locator(self.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        modal.get_by_role("button", name="Delete", exact=True).click()
        modal.wait_for(state="hidden", timeout=5000)

    def _select_multiple_items(
        self, items: list[Locator], then_right_click_last: bool = True
    ) -> None:
        """Select multiple items using Ctrl+Click.

        Args:
            items: List of Locator objects to select.
            then_right_click_last: If True, right-clicks the last item to open context menu.
        """
        for i, item in enumerate(items):
            if i == 0:
                item.click()
            else:
                item.click(modifiers=["ControlOrMeta"])

        if then_right_click_last and items:
            self.ctx_menu.open_on(items[-1])

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
