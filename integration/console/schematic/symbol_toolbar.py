#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Symbol Toolbar client for managing symbol groups and symbols."""

import json
from typing import Any

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.context_menu import ContextMenu
from console.layout import LayoutClient
from console.notifications import NotificationsClient
from console.schematic.symbol_editor import SymbolEditor
from framework.run_dir import resolve_results_path


class SymbolToolbar:
    """Client for interacting with the Schematic Symbols toolbar."""

    def __init__(self, layout: LayoutClient):
        self.page = layout.page
        self.layout = layout
        self.ctx_menu = ContextMenu(layout.page)
        self.notifications = NotificationsClient(layout.page)

    @property
    def toolbar(self) -> Locator:
        """Get the symbols toolbar locator."""
        return self.page.locator(".console-schematic__symbols")

    @property
    def group_list(self) -> Locator:
        """Get the group list container locator."""
        return self.page.locator(".console-schematic__symbols-group-list")

    def show(self) -> None:
        """Show the visualization toolbar with symbol search."""
        self.layout.show_visualization_toolbar()

    def _select_symbols_tab(self) -> None:
        """Select the Symbols tab.

        Dispatch the click: notifications stack over the drawer's tab strip
        and intercept physical clicks.
        """
        tab = self.page.get_by_text("Symbols", exact=True).first
        tab.wait_for(state="visible", timeout=5000)
        tab.dispatch_event("click")

    def _group_tab(self, name: str) -> Locator:
        """A symbol group's tab in the group list."""
        return self.group_list.locator("[role='tab']").filter(has_text=name)

    def select_group(self, name: str) -> None:
        """Select a symbol group by name."""
        self.show()
        self.notifications.close_all()
        self._select_symbols_tab()
        self.layout.wait_for_visible(self.group_list)

        group_btn = self._group_tab(name)
        self.layout.wait_for_visible(group_btn)
        self.layout.click(group_btn)

    def create_group(self, name: str) -> None:
        """Create a new symbol group via the toolbar button."""
        self.show()
        self.notifications.close_all()

        create_group_btn = (
            self.toolbar.locator("button[class*='outlined']")
            .filter(has=self.page.locator("[aria-label*='group']"))
            .first
        )
        # Notifications stack over the actions bar and swallow coordinate
        # clicks, so dispatch the click on the button itself.
        create_group_btn.wait_for(state="visible", timeout=5000)
        create_group_btn.dispatch_event("click")

        name_input = self.layout.locator("input[placeholder='Name']")
        self.layout.wait_for_visible(name_input)
        name_input.fill(name)

        self.layout.click_btn("Save")
        self.layout.wait_for_hidden(name_input)

        self.show()
        group_btn = self._group_tab(name)
        self.layout.wait_for_visible(group_btn)

    def _rename_inline(self, target: Locator, new_name: str) -> None:
        """Complete an in-place rename started from a context menu."""
        self.ctx_menu.action(target, "Rename")
        editable = self.page.locator(".pluto-text--editable[contenteditable='true']")
        editable.wait_for(state="visible", timeout=5000)
        editable.fill(new_name)
        self.page.keyboard.press("Enter")
        self.layout.wait_for_hidden(self.page.locator("[contenteditable='true']"))

    def rename_group(self, old_name: str, new_name: str) -> None:
        """Rename a symbol group via context menu (in-place edit)."""
        self.select_group(old_name)

        group_btn = self._group_tab(old_name)
        self._rename_inline(group_btn, new_name)

        renamed_btn = self._group_tab(new_name)
        self.layout.wait_for_visible(renamed_btn)

    def delete_group(self, name: str) -> None:
        """Delete a symbol group via context menu."""
        self.show()
        group_btn = self._group_tab(name)
        self.layout.wait_for_visible(group_btn)
        self.ctx_menu.action(group_btn, "Delete")

        confirm_btn = self.page.get_by_role("button", name="Delete")
        confirm_btn.wait_for(state="visible", timeout=3000)
        confirm_btn.click()
        confirm_btn.wait_for(state="hidden", timeout=3000)

        self.layout.wait_for_hidden(group_btn)

    def group_exists(self, name: str) -> bool:
        """Check if a symbol group exists."""
        try:
            self.show()
            self.layout.wait_for_visible(self.group_list)
            group_btn = self._group_tab(name)
            self.layout.wait_for_visible(group_btn)
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_for_group_hidden(self, name: str) -> None:
        """Wait for a symbol group to be hidden/removed."""
        group_btn = self._group_tab(name)
        self.layout.wait_for_hidden(group_btn)

    def create_symbol(self) -> SymbolEditor:
        """Open the symbol editor to create a new symbol."""
        create_symbol_btn = (
            self.toolbar.locator("button[class*='outlined']")
            .filter(has=self.page.locator("[aria-label*='schematic']"))
            .first
        )
        create_symbol_btn.click()

        editor = SymbolEditor(self.layout)
        editor.wait_for_open()
        return editor

    def get_symbol(self, name: str) -> Locator:
        """Get a symbol locator by name."""
        return self.toolbar.locator(".console-schematic-symbols__button").filter(
            has_text=name
        )

    def symbol_exists(self, name: str, select_group: str | None = None) -> bool:
        """Check if a symbol exists in the current group.

        Args:
            name: The symbol name to search for.
            select_group: Optional group name to select before searching.
        """
        if select_group is not None:
            self.select_group(select_group)
            symbol_container = self.toolbar.locator(
                ".console-schematic-symbols__button"
            )
            symbol_container.first.wait_for(state="attached", timeout=3000)

        symbol = self.get_symbol(name)
        try:
            symbol.wait_for(state="visible", timeout=3000)
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_for_symbol_hidden(self, name: str) -> None:
        """Wait for a symbol to be hidden/removed."""
        symbol = self.get_symbol(name)
        symbol.wait_for(state="hidden", timeout=5000)

    def rename_symbol(self, old_name: str, new_name: str) -> None:
        """Rename a symbol via context menu (in-place edit)."""
        symbol = self.get_symbol(old_name)
        self._rename_inline(symbol, new_name)

    def edit_symbol(self, name: str) -> SymbolEditor:
        """Open the symbol editor for an existing symbol via context menu."""
        symbol = self.get_symbol(name)
        self.ctx_menu.action(symbol, "Edit")

        editor = SymbolEditor(self.layout)
        editor.wait_for_open()
        return editor

    def delete_symbol(self, name: str) -> None:
        """Delete a symbol via context menu."""
        symbol = self.get_symbol(name)
        self.ctx_menu.action(symbol, "Delete")

        confirm_btn = self.page.get_by_role("button", name="Delete", exact=True)
        if confirm_btn.count() > 0:
            confirm_btn.click()

        symbol.wait_for(state="hidden", timeout=5000)

    def export_symbol(self, name: str) -> dict[str, Any]:
        """Export a symbol via context menu and return the JSON content."""
        symbol = self.get_symbol(name)
        self.ctx_menu.open_on(symbol)

        with self.page.expect_download(timeout=5000) as download_info:
            self.ctx_menu.click_option("Export")

        download = download_info.value
        save_path = resolve_results_path(f"{name}_export.json")
        download.save_as(save_path)

        with open(save_path, "r", encoding="utf-8") as f:
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
            group: Symbol group name (ignored - kept for backward compatibility).

        Returns:
            The data-testid of the newly created symbol node (e.g., "rf__node-{uuid}")
        """
        self.notifications.close_all()
        initial_count = len(self.page.locator("[data-testid^='rf__node-']").all())

        self._add_by_search(symbol_type)

        self.page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {initial_count}"
        )

        all_symbols = self.page.locator("[data-testid^='rf__node-']").all()
        return all_symbols[-1].get_attribute("data-testid") or "unknown"

    def _add_by_search(self, symbol_type: str) -> None:
        """Add a symbol using the search UI."""
        self.layout.show_visualization_toolbar()
        self._select_symbols_tab()
        self.layout.wait_for_visible(self.toolbar)

        search_input = self.toolbar.locator("input[role='textbox']").first
        search_input.wait_for(state="attached", timeout=5000)
        search_input.fill(symbol_type)

        self.layout.click(symbol_type)
