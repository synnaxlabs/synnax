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


class DevicesClient:
    """Console devices client for managing devices via the UI.

    Devices are shown in the Resources Toolbar under the devices panel (keyboard 'D').
    Each device shows its name, location, and status indicator.
    """

    ITEM_PREFIX = "device:"
    TOOLBAR_SELECTOR = ".console-nav__drawer"

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console

    def _show_devices_panel(self) -> None:
        """Show the devices panel in the navigation drawer."""
        devices_header = self.page.get_by_text("Devices", exact=True).first
        if devices_header.is_visible():
            return
        self.page.keyboard.press("d")
        devices_header.wait_for(state="visible", timeout=5000)
        self.page.wait_for_timeout(500)

    def _expand_rack(self, rack_name: str) -> None:
        """Expand a specific rack to show its devices."""
        rack_items = self.page.locator("div[id^='rack:']").filter(has_text=rack_name)
        if rack_items.count() == 0:
            return
        rack = rack_items.first
        expander = rack.locator("svg").first
        if expander.count() > 0:
            expander.click()
            self.page.wait_for_timeout(300)

    def _expand_all_racks(self) -> None:
        """Expand all rack items in the tree to show nested devices.

        Only expands racks that don't already have visible device children.
        """
        rack_items = self.page.locator("div[id^='rack:']")
        count = rack_items.count()

        all_devices = self.page.locator(f"div[id^='{self.ITEM_PREFIX}']")
        if all_devices.count() > 0 and all_devices.first.is_visible():
            return

        for i in range(count):
            rack = rack_items.nth(i)
            if not rack.is_visible():
                continue
            expander = rack.locator("svg").first
            if expander.count() > 0:
                expander.click()
                self.page.wait_for_timeout(100)
        self.page.wait_for_timeout(500)

    def debug_tree_contents(self) -> dict[str, list[str]]:
        """Get all racks and devices visible in the tree for debugging."""
        self._show_devices_panel()
        self._expand_all_racks()
        result = {"racks": [], "devices": []}

        rack_items = self.page.locator("div[id^='rack:']")
        for i in range(rack_items.count()):
            rack = rack_items.nth(i)
            if rack.is_visible():
                rack_id = rack.get_attribute("id") or ""
                text_el = rack.locator(".pluto-text--editable, .pluto-text").first
                name = text_el.inner_text().strip() if text_el.count() > 0 else ""
                result["racks"].append(f"{rack_id} ({name})")

        device_items = self.page.locator(f"div[id^='{self.ITEM_PREFIX}']")
        for i in range(device_items.count()):
            device = device_items.nth(i)
            if device.is_visible():
                device_id = device.get_attribute("id") or ""
                text_el = device.locator(".pluto-text--editable, .pluto-text").first
                name = text_el.inner_text().strip() if text_el.count() > 0 else ""
                result["devices"].append(f"{device_id} ({name})")

        return result

    def find_item(self, name: str, expand_tree: bool = True) -> Locator | None:
        """Find a device item in the devices panel by name.

        Args:
            name: The device name to find.
            expand_tree: Whether to expand collapsed rack items to find nested devices.
        """
        self._show_devices_panel()

        if expand_tree:
            self._expand_all_racks()

        self.page.wait_for_timeout(500)

        all_devices = self.page.locator(f"div[id^='{self.ITEM_PREFIX}']")
        items = all_devices.filter(has_text=name)
        if items.count() == 0:
            return None
        return items.first

    def get_item(self, name: str) -> Locator:
        """Get a device item locator from the devices panel."""
        item = self.find_item(name)
        if item is None:
            raise ValueError(f"Device '{name}' not found in devices panel")
        return item

    def exists(self, name: str) -> bool:
        """Check if a device exists in the devices panel."""
        return self.find_item(name) is not None

    def get_status(self, name: str) -> str | None:
        """Get the status indicator class of a device.

        Returns the status variant (e.g., 'success', 'error', 'warning', 'disabled')
        based on the status indicator's CSS classes.
        """
        self._show_devices_panel()
        device_item = self.get_item(name)
        status_indicator = device_item.locator(".pluto-device__status-indicator, .pluto-status__indicator").first
        if status_indicator.count() == 0:
            return None
        class_attr = status_indicator.get_attribute("class") or ""
        if "success" in class_attr:
            return "success"
        elif "error" in class_attr:
            return "error"
        elif "warning" in class_attr:
            return "warning"
        return "disabled"

    def wait_for_status(
        self, name: str, expected_status: str, timeout: int = 10000
    ) -> bool:
        """Wait for a device to reach a specific status.

        Args:
            name: The device name.
            expected_status: The expected status ('success', 'error', 'warning', 'disabled').
            timeout: Maximum time to wait in milliseconds.

        Returns:
            True if the status was reached, False if timeout occurred.
        """
        import time

        start = time.time()
        timeout_sec = timeout / 1000
        while time.time() - start < timeout_sec:
            current_status = self.get_status(name)
            if current_status == expected_status:
                return True
            self.page.wait_for_timeout(500)
        return False

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename a device via context menu."""
        self._show_devices_panel()
        device_item = self.get_item(old_name)
        device_item.click(button="right")
        self.page.get_by_text("Rename", exact=True).click(timeout=2000)
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type(new_name)
        self.page.keyboard.press("Enter")
        self.page.wait_for_timeout(500)

    def delete(self, name: str) -> None:
        """Delete a device via context menu."""
        self._show_devices_panel()
        device_item = self.get_item(name)
        device_item.click(button="right")
        self.page.get_by_text("Delete", exact=True).click(timeout=2000)
        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=3000)
        delete_btn.click()
        delete_btn.wait_for(state="hidden", timeout=3000)

    def group(self, device_names: list[str], group_name: str | None = None) -> None:
        """Group multiple devices via context menu.

        Args:
            device_names: List of device names to group.
            group_name: Optional name for the new group. If not provided, uses default.
        """
        self._show_devices_panel()
        self._expand_all_racks()

        first_device = self.get_item(device_names[0])
        first_device.click()
        self.page.wait_for_timeout(200)

        for device_name in device_names[1:]:
            device_item = self.get_item(device_name)
            self.console.meta_click(device_item)

        self.page.wait_for_timeout(200)
        first_device.click(button="right")
        self.page.wait_for_timeout(300)

        group_option = self.page.get_by_text("Group", exact=True)
        if group_option.is_visible():
            group_option.click(timeout=2000)

            if group_name is not None:
                self.page.wait_for_timeout(300)
                self.page.keyboard.press("ControlOrMeta+a")
                self.page.keyboard.type(group_name)
                self.page.keyboard.press("Enter")
            else:
                self.page.keyboard.press("Enter")
        else:
            self.page.keyboard.press("Escape")

        self.page.wait_for_timeout(500)

    def configure(self, name: str) -> None:
        """Open the configuration dialog for an unconfigured device via context menu.

        This option is only available for devices that have not been configured yet.
        """
        self._show_devices_panel()
        device_item = self.get_item(name)
        device_item.click(button="right")
        configure_option = self.page.get_by_text("Configure", exact=True)
        configure_option.wait_for(state="visible", timeout=2000)
        configure_option.click()
        self.page.wait_for_timeout(500)

    def change_identifier(self, name: str, new_identifier: str) -> None:
        """Change the identifier of a configured device via context menu.

        This option is only available for configured LabJack and NI devices.

        Args:
            name: The device name.
            new_identifier: The new identifier to set.
        """
        self._show_devices_panel()
        device_item = self.get_item(name)
        device_item.click(button="right")
        change_id_option = self.page.get_by_text("Change identifier", exact=True)
        change_id_option.wait_for(state="visible", timeout=2000)
        change_id_option.click()

        rename_input = self.page.locator("input.pluto-rename__input")
        rename_input.wait_for(state="visible", timeout=3000)
        rename_input.fill(new_identifier)
        self.page.keyboard.press("Enter")
        self.page.wait_for_timeout(500)

    def is_configured(self, name: str) -> bool:
        """Check if a device is configured by checking available context menu options.

        A configured device shows "Change identifier" (for LabJack/NI) instead of "Configure".
        """
        self._show_devices_panel()
        device_item = self.get_item(name)
        device_item.click(button="right")
        self.page.wait_for_timeout(300)

        configure_visible = (
            self.page.get_by_text("Configure", exact=True).is_visible()
        )
        self.page.keyboard.press("Escape")

        return not configure_visible

    def list_all(self) -> list[str]:
        """List all device names in the devices panel."""
        self._show_devices_panel()
        device_items = self.page.locator(f"div[id^='{self.ITEM_PREFIX}']").all()
        names = []
        for item in device_items:
            if item.is_visible():
                text_el = item.locator(".pluto-text--editable, .pluto-text").first
                if text_el.count() > 0:
                    name = text_el.inner_text().strip()
                    if name:
                        names.append(name)
        return names

    def get_location(self, name: str) -> str | None:
        """Get the location metadata of a device."""
        self._show_devices_panel()
        device_item = self.get_item(name)
        location_el = device_item.locator(".console-device-ontology-item__location, .location").first
        if location_el.count() == 0:
            return None
        return location_el.inner_text().strip()
