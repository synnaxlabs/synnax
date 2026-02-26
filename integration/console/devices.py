#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import re

import synnax as sy
from playwright.sync_api import Locator

from console.context_menu import ContextMenu
from console.layout import LayoutClient
from console.tree import Tree


class DevicesClient:
    """Device and rack management for Console UI automation."""

    RACK_PREFIX = "rack:"
    DEVICE_PREFIX = "device:"
    ICON_NAME = "device"

    def __init__(self, layout: LayoutClient, client: sy.Synnax):
        self.layout = layout
        self.client = client
        self.ctx_menu = ContextMenu(layout.page)
        self.tree = Tree(layout.page)

    # ── Panel Visibility ─────────────────────────────────────────────────

    def show_toolbar(self) -> None:
        """Show the devices toolbar in the left sidebar."""
        self.layout.show_resource_toolbar(self.ICON_NAME)

    # ── Shared Helpers ────────────────────────────────────────────────────

    def _find_item(self, prefix: str, name: str) -> Locator | None:
        """Find an item in the tree by prefix and name."""
        self.show_toolbar()
        return self.tree.find_by_name(prefix, name, exact=False)

    def _get_item(self, prefix: str, name: str, label: str = "Item") -> Locator:
        """Get an item locator, raising if not found."""
        item = self._find_item(prefix, name)
        if item is None:
            raise ValueError(f"{label} '{name}' not found")
        return item

    def _modal_fill_and_submit(
        self,
        action: str,
        device_item: Locator,
        steps: list[tuple[str, str]],
    ) -> None:
        """Open a context menu action and fill modal steps.

        :param action: Context menu action text (e.g. "Configure").
        :param device_item: The tree item locator to right-click.
        :param steps: List of (input_value, button_text) pairs for each
            modal step.
        """
        self.ctx_menu.action(device_item, action)
        modal = self.layout.page.locator(self.layout.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        for value, button in steps:
            input_el = modal.locator("input").first
            input_el.wait_for(state="visible", timeout=5000)
            input_el.fill(value)
            modal.get_by_role("button", name=button, exact=True).dispatch_event("click")
        modal.wait_for(state="hidden", timeout=5000)

    # ── Rack Methods ─────────────────────────────────────────────────────

    def find_rack(self, name: str) -> Locator | None:
        """Find a rack item in the devices panel by name."""
        return self._find_item(self.RACK_PREFIX, name)

    def get_rack(self, name: str) -> Locator:
        """Get a rack item locator from the devices panel."""
        return self._get_item(self.RACK_PREFIX, name, "Rack")

    def rack_exists(self, name: str) -> bool:
        """Check if a rack exists in the devices panel."""
        return self._find_item(self.RACK_PREFIX, name) is not None

    def wait_for_rack_removed(self, name: str) -> None:
        """Wait for a rack to be removed from the devices panel."""
        self.show_toolbar()
        self.tree.wait_for_removal(self.RACK_PREFIX, name, exact=False)

    def get_rack_status(self, name: str) -> dict[str, str]:
        """Get the status of a rack by hovering over its status indicator."""
        self.show_toolbar()
        rack_item = self.get_rack(name)
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

    def rename_rack(self, *, old_name: str, new_name: str) -> None:
        """Rename a rack via context menu."""
        self.show_toolbar()
        rack_item = self.get_rack(old_name)
        self.layout.context_menu_action(rack_item, "Rename")
        self.layout.select_all_and_type(new_name)
        self.layout.press_enter()
        new_item = self.layout.page.locator(f"div[id^='{self.RACK_PREFIX}']").filter(
            has_text=new_name
        )
        new_item.first.wait_for(state="visible", timeout=5000)
        self.wait_for_rack_removed(old_name)

    def delete_rack(self, name: str) -> None:
        """Delete a rack via context menu."""
        self.show_toolbar()
        rack_item = self.get_rack(name)
        self.layout.delete_with_confirmation(rack_item)
        self.wait_for_rack_removed(name)

    def copy_rack_key(self, name: str) -> str:
        """Copy a rack's key to clipboard via context menu."""
        self.show_toolbar()
        rack_item = self.get_rack(name)
        element_id = rack_item.get_attribute("id")
        rack_key = element_id.split(":")[1] if element_id else ""
        self.layout.context_menu_action(rack_item, "Copy properties")
        return rack_key

    # ── Device Methods ───────────────────────────────────────────────────

    def _ensure_device_visible(self, name: str) -> Locator | None:
        """Expand ancestor nodes so a device becomes visible in the tree.

        Uses the Python client to look up the device's ancestry, then
        expands nodes in order: rack → groups → parent chassis → groups.

        :param name: Name of the device.
        :returns: The Locator if the device is now visible, None otherwise.
        """
        self.show_toolbar()
        item = self.tree.find_by_name(self.DEVICE_PREFIX, name, exact=False)
        if item is not None:
            return item
        device = self.client.devices.retrieve(name=name, ignore_not_found=True)
        if device is None:
            return None
        # Expand the rack that owns this device, waiting for it to appear.
        rack = self.client.racks.retrieve(device.rack)
        rack_item = self.tree.find_by_name(self.RACK_PREFIX, rack.name, exact=False)
        if rack_item is None:
            self.layout.page.locator(f"div[id^='{self.RACK_PREFIX}']").filter(
                has_text=rack.name
            ).first.wait_for(state="visible", timeout=10000)
            rack_item = self.tree.find_by_name(self.RACK_PREFIX, rack.name, exact=False)
        if rack_item is not None:
            self.tree.expand(rack_item)
        self._expand_visible_groups()
        # If the device has a parent chassis, expand that too.
        if device.parent_device:
            parent = self.client.devices.retrieve(key=device.parent_device)
            parent_item = self.tree.find_by_name(
                self.DEVICE_PREFIX, parent.name, exact=False
            )
            if parent_item is not None:
                self.tree.expand(parent_item)
        self._expand_visible_groups()
        return self.tree.find_by_name(self.DEVICE_PREFIX, name, exact=False)

    def _expand_visible_groups(self) -> None:
        """Expand all visible collapsed group nodes."""
        groups = self.layout.page.locator("div[id^='group:']").all()
        for g in groups:
            if g.is_visible() and not self.tree.is_expanded(g):
                self.tree.expand(g)

    def find(self, name: str) -> Locator | None:
        """Find a device item in the devices panel by name."""
        return self._ensure_device_visible(name)

    def get(self, name: str) -> Locator:
        """Get a device item locator from the devices panel."""
        item = self._ensure_device_visible(name)
        if item is None:
            raise ValueError(f"Device '{name}' not found")
        return item

    def exists(self, name: str) -> bool:
        """Check if a device exists in the devices panel."""
        return self._ensure_device_visible(name) is not None

    def wait_for(self, name: str, *, timeout: int = 10000) -> None:
        """Wait for a device to appear in the devices panel."""
        self.show_toolbar()
        self.layout.page.locator(f"div[id^='{self.DEVICE_PREFIX}']").filter(
            has_text=name
        ).first.wait_for(state="visible", timeout=timeout)

    def wait_for_removed(self, name: str) -> None:
        """Wait for a device to be removed from the devices panel."""
        self.show_toolbar()
        self.tree.wait_for_removal(self.DEVICE_PREFIX, name, exact=False)

    # Map CSS color variables to status variants.
    _COLOR_TO_VARIANT: dict[str, str] = {
        "--pluto-primary-z": "success",
        "--pluto-gray-l8": "disabled",
        "--pluto-warning-m1": "warning",
        "--pluto-error-m1": "error",
    }

    def get_status(self, name: str) -> dict[str, str]:
        """Get the status of a device by hovering over its status indicator.

        The device status indicator is a colored circle SVG with class
        ``pluto-device__status-indicator``.  The variant is determined from
        the ``color`` inline style which uses a CSS variable
        (e.g. ``var(--pluto-warning-m1)``).

        :returns: Dict with 'variant' (str) and 'message' (str).
        """
        self.show_toolbar()
        device_item = self.get(name)
        status_icon = device_item.locator("svg.pluto-device__status-indicator").first
        status_icon.wait_for(state="visible", timeout=2000)
        status_icon.hover()
        tooltip = self.layout.page.locator(".pluto-tooltip")
        tooltip.wait_for(state="visible", timeout=3000)
        message = tooltip.inner_text().strip()
        color_attr = status_icon.get_attribute("color") or ""
        variant = "disabled"
        for css_var, v in self._COLOR_TO_VARIANT.items():
            if css_var in color_attr:
                variant = v
                break
        self.layout.page.mouse.move(0, 0)
        return {"variant": variant, "message": message}

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename a device via context menu inline edit."""
        self.show_toolbar()
        device_item = self.get(old_name)
        self.tree.rename(device_item, new_name)
        self.layout.page.locator(f"div[id^='{self.DEVICE_PREFIX}']").filter(
            has_text=new_name
        ).first.wait_for(state="visible", timeout=5000)

    def delete(self, name: str) -> None:
        """Delete a single device via context menu with confirmation."""
        self.show_toolbar()
        device_item = self.get(name)
        self.layout.delete_with_confirmation(device_item)
        self.wait_for_removed(name)

    def delete_multi(self, names: list[str]) -> None:
        """Multi-select devices and delete via context menu."""
        self.show_toolbar()
        items: list[Locator] = []
        for name in names:
            item = self.get(name)
            items.append(item)
        if not items:
            return
        items[0].click()
        for item in items[1:]:
            item.click(modifiers=["ControlOrMeta"])
        self.ctx_menu.action(items[-1], "Delete")
        self.layout.confirm_delete()
        for name in names:
            self.wait_for_removed(name)

    def group(self, names: list[str], group_name: str) -> None:
        """Group devices via multi-select and 'Group selection' context menu.

        :param names: Device names to group.
        :param group_name: Name for the new group.
        """
        self.show_toolbar()
        items: list[Locator] = []
        for name in names:
            item = self.tree.find_by_name(self.DEVICE_PREFIX, name, exact=False)
            if item is None:
                item = self.tree.find_by_name(self.RACK_PREFIX, name, exact=False)
            if item is None:
                raise ValueError(f"Item '{name}' not found in devices panel")
            items.append(item)
        self.tree.group(items, group_name)

    def _set_chassis_expanded(self, name: str, *, expanded: bool) -> None:
        """Expand or collapse a chassis device node."""
        self.show_toolbar()
        chassis = self.get(name)
        is_expanded = self.tree.is_expanded(chassis)
        if expanded and not is_expanded:
            self.tree.expand(chassis)
        elif not expanded and is_expanded:
            self.tree.collapse(chassis)

    def expand_chassis(self, name: str) -> None:
        """Expand a chassis device node to reveal its children."""
        self._set_chassis_expanded(name, expanded=True)

    def collapse_chassis(self, name: str) -> None:
        """Collapse a chassis device node."""
        self._set_chassis_expanded(name, expanded=False)

    def get_children_names(self, chassis_name: str) -> list[str]:
        """Get the visible child device names under a chassis.

        Expands the chassis first, then collects all device-prefixed children
        by walking sibling device items with greater tree depth.  Retries up
        to three times to handle transient tree re-renders (e.g. after a
        rename triggers a server-side refresh).

        :param chassis_name: Name of the chassis device.
        :returns: List of child device names.
        """
        self.show_toolbar()
        self.expand_chassis(chassis_name)
        names = self._scan_children(chassis_name)
        if names:
            return names

        # Retry logic because test cases run fast.
        for _ in range(5):
            names = self._scan_children(chassis_name)
            if names:
                return names
            self.collapse_chassis(chassis_name)
            self.expand_chassis(chassis_name)
        return []

    def _scan_children(self, chassis_name: str) -> list[str]:
        """Scan device items after the chassis and collect those at greater depth."""
        chassis = self.get(chassis_name)
        all_device_items = self.layout.page.locator(
            f"div[id^='{self.DEVICE_PREFIX}']"
        ).all()
        names: list[str] = []
        found_chassis = False
        chassis_depth = self._get_depth(chassis)
        for item in all_device_items:
            if not item.is_visible():
                continue
            item_text = self.tree.get_text(item)
            if item_text == chassis_name or chassis_name in item_text:
                found_chassis = True
                continue
            if found_chassis:
                item_depth = self._get_depth(item)
                if item_depth > chassis_depth:
                    names.append(self.tree.get_text(item))
                else:
                    break
        return names

    def _get_depth(self, item: Locator) -> float:
        """Get the tree depth offset of an item from its CSS variable.

        The Pluto tree sets ``--pluto-tree-item-offset`` as an inline
        style (e.g. ``--pluto-tree-item-offset: 6.5rem``).  We parse
        the numeric value to compare parent/child relationships.
        """
        style = item.get_attribute("style") or ""
        match = re.search(r"--pluto-tree-item-offset:\s*([\d.]+)", style)
        return float(match.group(1)) if match else 0.0

    def is_child_of(self, device_name: str, parent_name: str) -> bool:
        """Check if a device is nested under a parent in the tree.

        :param device_name: Name of the child device.
        :param parent_name: Name of the parent device or chassis.
        :returns: True if the device appears under the parent.
        """
        children = self.get_children_names(parent_name)
        return any(device_name in child for child in children)

    def get_icon(self, name: str) -> str | None:
        """Get the icon type rendered for a device in the tree.

        The icon's ``aria-label`` follows the pattern ``pluto-icon--logo-{make}``.

        :param name: Name of the device.
        :returns: The make slug (e.g. ``"ni"``, ``"labjack"``), or ``None``
            if no icon is present.
        """
        device_item = self.get(name)
        icon = device_item.locator(
            "svg.pluto-icon:not(.pluto-tree__expansion-indicator)"
        ).first
        if icon.count() == 0:
            return None
        aria = icon.get_attribute("aria-label") or ""
        prefix = "pluto-icon--logo-"
        idx = aria.find(prefix)
        if idx == -1:
            return None
        return aria[idx + len(prefix) :]

    def configure(self, name: str, *, device_name: str, identifier: str) -> None:
        """Configure an unconfigured device via the two-step modal flow.

        Step 1: Enter a device name and click "Next".
        Step 2: Enter an identifier and click "Save".

        :param name: Current display name of the device in the tree.
        :param device_name: Name to enter in the first modal step.
        :param identifier: Identifier to enter in the second modal step.
        """
        self.show_toolbar()
        self._modal_fill_and_submit(
            "Configure",
            self.get(name),
            [(device_name, "Next"), (identifier, "Save")],
        )

    def copy_properties(self, name: str) -> dict[str, object]:
        """Copy a device's properties to the clipboard and return as a dict.

        :param name: Name of the device.
        :returns: Parsed JSON properties dict.
        """
        self.show_toolbar()
        device_item = self.get(name)
        self.ctx_menu.action(device_item, "Copy properties")
        self.layout.page.wait_for_timeout(300)
        raw = self.layout.read_clipboard()
        result: dict[str, object] = json.loads(raw)
        return result

    def change_identifier(self, name: str, new_identifier: str) -> None:
        """Change a configured device's identifier via context menu modal.

        :param name: Name of the configured device.
        :param new_identifier: The new identifier to set.
        """
        self.show_toolbar()
        self._modal_fill_and_submit(
            "Change identifier",
            self.get(name),
            [(new_identifier, "Save")],
        )
