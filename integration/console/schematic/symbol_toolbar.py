#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Symbol Toolbar client for managing symbol groups and symbols."""

from __future__ import annotations

import json
from typing import TYPE_CHECKING, Any

from playwright.sync_api import Locator, Page

from framework.utils import get_results_path

if TYPE_CHECKING:
    from console.console import Console
    from console.schematic.symbol_editor import SymbolEditor


class SymbolToolbar:
    """Client for interacting with the Schematic Symbols toolbar."""

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console

    @property
    def toolbar(self) -> Locator:
        """Get the symbols toolbar locator."""
        return self.page.locator(".console-schematic__symbols")

    @property
    def group_list(self) -> Locator:
        """Get the group list locator."""
        return self.page.locator(".console-schematic__symbols-group-list")

    def show(self) -> None:
        """Show the Symbols tab in the schematic toolbar."""
        symbols_tab = self.page.get_by_text("Symbols", exact=True).first
        symbols_tab.click()
        self.toolbar.wait_for(state="visible", timeout=5000)

    def select_group(self, name: str) -> None:
        """Select a symbol group by name."""
        self.show()
        group_btn = self.group_list.locator(".pluto-btn").filter(has_text=name)
        group_btn.wait_for(state="visible", timeout=5000)
        group_btn.click()

    def create_group(self, name: str) -> None:
        """Create a new symbol group via the toolbar button."""
        self.show()
        create_group_btn = (
            self.toolbar.locator("button[class*='outlined']")
            .filter(has=self.page.locator("[aria-label*='group']"))
            .first
        )
        self.console.notifications.close_all()
        create_group_btn.click()

        name_input = self.page.locator("input[placeholder='Group Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)

        save_btn = self.page.get_by_role("button", name="Save", exact=True)
        save_btn.click()
        name_input.wait_for(state="hidden", timeout=5000)

        group_btn = self.group_list.locator(".pluto-btn").filter(has_text=name)
        group_btn.wait_for(state="visible", timeout=10000)

    def rename_group(self, old_name: str, new_name: str) -> None:
        """Rename a symbol group via context menu."""
        self.select_group(old_name)
        group_btn = self.group_list.locator(".pluto-btn").filter(has_text=old_name)
        group_btn.click(button="right")

        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        menu.get_by_text("Rename", exact=True).click()

        name_input = self.page.locator("input[placeholder='Group Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(new_name)

        save_btn = self.page.get_by_role("button", name="Save", exact=True)
        save_btn.click()
        name_input.wait_for(state="hidden", timeout=5000)

        renamed_btn = self.group_list.locator(".pluto-btn").filter(has_text=new_name)
        renamed_btn.wait_for(state="visible", timeout=5000)

    def delete_group(self, name: str) -> None:
        """Delete a symbol group via context menu."""
        group_btn = self.group_list.locator(".pluto-btn").filter(has_text=name)
        group_btn.click(button="right")

        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        menu.get_by_text("Delete", exact=True).click()

        confirm_btn = self.page.get_by_role("button", name="Delete", exact=True)
        if confirm_btn.count() > 0:
            confirm_btn.click()
            confirm_btn.wait_for(state="hidden", timeout=5000)

        group_btn.wait_for(state="hidden", timeout=5000)

    def group_exists(self, name: str) -> bool:
        """Check if a symbol group exists."""
        try:
            self.group_list.wait_for(state="visible", timeout=3000)
            group_btn = self.group_list.locator(".pluto-btn").filter(has_text=name)
            group_btn.wait_for(state="visible", timeout=3000)
            return True
        except Exception as e:
            if "Timeout" in type(e).__name__:
                return False
            raise RuntimeError(f"Error checking if group '{name}' exists: {e}") from e

    def wait_for_group_hidden(self, name: str) -> None:
        """Wait for a symbol group to be hidden/removed."""
        group_btn = self.group_list.locator(".pluto-btn").filter(has_text=name)
        group_btn.wait_for(state="hidden", timeout=5000)

    def create_symbol(self) -> "SymbolEditor":
        """Open the symbol editor to create a new symbol."""
        from console.schematic.symbol_editor import SymbolEditor

        create_symbol_btn = (
            self.toolbar.locator("button[class*='outlined']")
            .filter(has=self.page.locator("[aria-label*='schematic']"))
            .first
        )
        create_symbol_btn.click()

        editor = SymbolEditor(self.page, self.console)
        editor.wait_for_open()
        return editor

    def get_symbol(self, name: str) -> Locator:
        """Get a symbol locator by name."""
        return self.toolbar.locator(".console-schematic-symbols__button").filter(
            has_text=name
        )

    def symbol_exists(self, name: str) -> bool:
        """Check if a symbol exists in the current group."""
        try:
            symbol = self.get_symbol(name)
            symbol.wait_for(state="visible", timeout=3000)
            return True
        except Exception as e:
            if "Timeout" in type(e).__name__:
                return False
            raise RuntimeError(f"Error checking if symbol '{name}' exists: {e}") from e

    def wait_for_symbol_hidden(self, name: str) -> None:
        """Wait for a symbol to be hidden/removed."""
        symbol = self.get_symbol(name)
        symbol.wait_for(state="hidden", timeout=5000)

    def rename_symbol(self, old_name: str, new_name: str) -> None:
        """Rename a symbol via context menu."""
        symbol = self.get_symbol(old_name)
        symbol.click(button="right")

        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        menu.get_by_text("Rename", exact=True).click()

        name_input = self.page.locator("input[placeholder='Symbol Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(new_name)

        save_btn = self.page.get_by_role("button", name="Save", exact=True)
        save_btn.click()
        name_input.wait_for(state="hidden", timeout=5000)

    def edit_symbol(self, name: str) -> "SymbolEditor":
        """Open the symbol editor for an existing symbol via context menu."""
        from console.schematic.symbol_editor import SymbolEditor

        symbol = self.get_symbol(name)
        symbol.click(button="right")

        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        menu.get_by_text("Edit", exact=True).click()

        editor = SymbolEditor(self.page, self.console)
        editor.wait_for_open()
        return editor

    def delete_symbol(self, name: str) -> None:
        """Delete a symbol via context menu."""
        symbol = self.get_symbol(name)
        symbol.click(button="right")

        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        menu.get_by_text("Delete", exact=True).click()

        confirm_btn = self.page.get_by_role("button", name="Delete", exact=True)
        if confirm_btn.count() > 0:
            confirm_btn.click()

        symbol.wait_for(state="hidden", timeout=5000)

    def export_symbol(self, name: str) -> dict[str, Any]:
        """Export a symbol via context menu and return the JSON content."""
        symbol = self.get_symbol(name)
        symbol.click(button="right")

        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)

        self.page.evaluate("delete window.showSaveFilePicker")

        with self.page.expect_download(timeout=5000) as download_info:
            menu.get_by_text("Export", exact=True).click()

        download = download_info.value
        save_path = get_results_path(f"{name}_export.json")
        download.save_as(save_path)

        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def drag_symbol_to_schematic(self, name: str) -> None:
        """Drag a symbol onto the schematic canvas."""
        symbol = self.get_symbol(name)
        canvas = self.page.locator(".react-flow__pane").first
        symbol.drag_to(canvas)

    def add_symbol(self, symbol_type: str, group: str | None = None) -> str:
        """Add any symbol to the schematic and return its ID.

        This is the universal factory method for adding both built-in and custom
        symbols to the schematic canvas.

        Args:
            symbol_type: Name of the symbol (e.g., "Button", "Valve", "My Custom Pump")
            group: Symbol group name. If provided, selects the group and drags.
                   If None, uses search to find and add the symbol.

        Returns:
            The data-testid of the newly created symbol node (e.g., "rf__node-{uuid}")
        """
        self.console.notifications.close_all()
        self.show()
        initial_count = len(self.page.locator("[data-testid^='rf__node-']").all())

        if group is not None:
            self.select_group(group)
            self.toolbar.locator(".console-schematic-symbols__button").first.wait_for(
                state="visible", timeout=10000
            )
            symbol = self.get_symbol(symbol_type)
            canvas = self.page.locator(".react-flow__pane").first
            symbol.drag_to(canvas)
        else:
            self._add_by_search(symbol_type)

        self.page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {initial_count}"
        )

        all_symbols = self.page.locator("[data-testid^='rf__node-']").all()
        return all_symbols[-1].get_attribute("data-testid") or "unknown"

    def _add_by_search(self, symbol_type: str) -> None:
        """Add a symbol using the search UI."""
        search_input = self.page.locator(
            "div:has-text('Search Symbols') input[role='textbox']"
        ).first
        search_input.fill(symbol_type)
        self.console.click(symbol_type)
