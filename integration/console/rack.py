#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator

from .base import BaseClient
from .layout import LayoutClient


class RackClient(BaseClient):
    """Rack management for Console UI automation."""

    ITEM_PREFIX = "rack:"
    SHORTCUT_KEY = "d"

    def __init__(self, layout: LayoutClient):
        super().__init__(layout)

    def _show_devices_panel(self) -> None:
        """Show the devices panel in the navigation drawer."""
        self._show_toolbar(self.SHORTCUT_KEY, self.ITEM_PREFIX)

    def find_item(self, name: str) -> Locator | None:
        """Find a rack item in the devices panel by name."""
        self._show_devices_panel()
        return self._find_toolbar_item(self.ITEM_PREFIX, name)

    def get_item(self, name: str) -> Locator:
        """Get a rack item locator from the devices panel."""
        self._show_devices_panel()
        return self._get_toolbar_item(self.ITEM_PREFIX, name)

    def exists(self, name: str) -> bool:
        """Check if a rack exists in the devices panel."""
        self._show_devices_panel()
        return self._toolbar_item_exists(self.ITEM_PREFIX, name)

    def wait_for_rack_removed(self, name: str, timeout: int = 5000) -> None:
        """Wait for a rack to be removed from the devices panel.

        Args:
            name: Name of the rack to wait for removal
            timeout: Maximum time in milliseconds to wait
        """
        self._show_devices_panel()
        self._wait_for_item_removed(self.ITEM_PREFIX, name, timeout)

    def get_status(self, name: str) -> dict[str, str]:
        """Get the status of a rack by hovering over its status indicator."""
        self._show_devices_panel()
        rack_item = self.get_item(name)
        status_icon = rack_item.locator("svg.pluto-rack__heartbeat")
        status_icon.wait_for(state="visible", timeout=2000)
        status_icon.hover()
        tooltip = self.layout.page.locator(".pluto-tooltip")
        tooltip.wait_for(state="visible", timeout=3000)
        message = tooltip.inner_text().strip()
        class_attr = status_icon.get_attribute("class") or ""
        if "pluto-rack__heartbeat--beat" in class_attr:
            variant = "success"
        else:
            variant = "disabled"
        self.layout.page.mouse.move(0, 0)
        return {"variant": variant, "message": message}

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename a rack via context menu."""
        self._show_devices_panel()
        rack_item = self.get_item(old_name)
        self._context_menu_action(rack_item, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        new_item = self.layout.page.locator(f"div[id^='{self.ITEM_PREFIX}']").filter(
            has_text=new_name
        )
        new_item.first.wait_for(state="visible", timeout=5000)
        self.wait_for_rack_removed(old_name)

    def delete(self, name: str) -> None:
        """Delete a rack via context menu."""
        self._show_devices_panel()
        rack_item = self.get_item(name)
        self._delete_with_confirmation(rack_item)
        self.wait_for_rack_removed(name)

    def copy_key(self, name: str) -> str:
        """Copy a rack's key to clipboard via context menu."""
        self._show_devices_panel()
        rack_item = self.get_item(name)
        element_id = rack_item.get_attribute("id")
        rack_key = element_id.split(":")[1] if element_id else ""
        self._context_menu_action(rack_item, "Copy properties")
        return rack_key
