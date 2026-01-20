#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING

from playwright.sync_api import Locator, Page

if TYPE_CHECKING:
    from .console import Console


class RackClient:
    """Rack management for Console UI automation."""

    ITEM_PREFIX = "rack:"

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console

    def _show_devices_panel(self) -> None:
        """Show the devices panel in the navigation drawer."""
        rack_elements = self.page.locator(f"div[id^='{self.ITEM_PREFIX}']")
        if rack_elements.count() > 0 and rack_elements.first.is_visible():
            return
        self.page.keyboard.press("d")
        self.page.locator(f"div[id^='{self.ITEM_PREFIX}']").first.wait_for(
            state="visible", timeout=5000
        )

    def find_item(self, name: str) -> Locator | None:
        """Find a rack item in the devices panel by name."""
        self._show_devices_panel()
        items = self.page.locator(f"div[id^='{self.ITEM_PREFIX}']").filter(
            has_text=name
        )
        if items.count() == 0:
            return None
        return items.first

    def get_item(self, name: str) -> Locator:
        """Get a rack item locator from the devices panel."""
        item = self.find_item(name)
        if item is None:
            raise ValueError(f"Rack '{name}' not found in devices panel")
        return item

    def exists(self, name: str) -> bool:
        """Check if a rack exists in the devices panel."""
        return self.find_item(name) is not None

    def get_status(self, name: str) -> dict[str, str]:
        """Get the status of a rack by hovering over its status indicator."""
        self._show_devices_panel()
        rack_item = self.get_item(name)
        status_icon = rack_item.locator("svg.pluto-rack__heartbeat")
        status_icon.wait_for(state="visible", timeout=2000)
        status_icon.hover()
        tooltip = self.page.locator(".pluto-tooltip")
        tooltip.wait_for(state="visible", timeout=3000)
        message = tooltip.inner_text().strip()
        class_attr = status_icon.get_attribute("class") or ""
        if "pluto-rack__heartbeat--beat" in class_attr:
            variant = "success"
        else:
            variant = "disabled"
        self.page.mouse.move(0, 0)
        return {"variant": variant, "message": message}

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename a rack via context menu."""
        self._show_devices_panel()
        rack_item = self.get_item(old_name)
        rack_item.click(button="right")
        self.page.get_by_text("Rename", exact=True).click(timeout=2000)
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type(new_name)
        self.page.keyboard.press("Enter")

    def delete(self, name: str) -> None:
        """Delete a rack via context menu."""
        self._show_devices_panel()
        rack_item = self.get_item(name)
        rack_item.click(button="right")
        self.page.get_by_text("Delete", exact=True).click(timeout=2000)
        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=3000)
        delete_btn.click()

    def copy_key(self, name: str) -> str:
        """Copy a rack's key to clipboard via context menu."""
        self._show_devices_panel()
        rack_item = self.get_item(name)
        element_id = rack_item.get_attribute("id")
        rack_key = element_id.split(":")[1] if element_id else ""
        rack_item.click(button="right")
        self.page.get_by_text("Copy properties", exact=True).click(timeout=2000)
        return rack_key
