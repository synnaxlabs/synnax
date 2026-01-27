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


class ArcClient:
    """Arc automation management for Console UI automation."""

    TOOLBAR_CLASS = ".console-arc-toolbar"
    CONTROLS_CLASS = ".console-arc-editor__controls"
    LIST_ITEM_CLASS = ".pluto-list__item"

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console

    def _show_arc_panel(self) -> None:
        """Show the arc panel in the navigation drawer."""
        self.console.ESCAPE
        toolbar = self.page.locator(self.TOOLBAR_CLASS)
        if toolbar.count() > 0 and toolbar.is_visible():
            return
        arc_btn = self.page.locator("button:has(.pluto-icon--arc)")
        arc_btn.click()
        toolbar.wait_for(state="visible", timeout=5000)

    def _get_controls(self) -> Locator:
        """Get the Arc editor controls locator."""
        controls = self.page.locator(self.CONTROLS_CLASS)
        controls.wait_for(state="visible", timeout=5000)
        return controls

    def create(self, name: str, source: str, mode: str = "Text") -> None:
        """Create a new Arc via Console UI."""
        self._show_arc_panel()

        add_btn = self.page.locator(f"{self.TOOLBAR_CLASS} button").first
        add_btn.click()

        name_input = self.page.locator("input[placeholder='Automation Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)

        mode_btn = self.page.locator(
            f".console-arc-create-modal__mode-select-button:has-text('{mode}')"
        )
        mode_btn.wait_for(state="visible", timeout=5000)
        mode_btn.click()

        create_btn = self.page.get_by_role("button", name="Create", exact=True)
        create_btn.wait_for(state="visible", timeout=5000)
        create_btn.click()
        name_input.wait_for(state="hidden", timeout=10000)

        if mode == "Text":
            editor = self.page.locator("[data-mode-id='arc']")
            editor.wait_for(state="visible", timeout=10000)
            editor.click()
            editor.locator(".cursor").wait_for(state="visible", timeout=5000)
            self.page.keyboard.press("ControlOrMeta+a")
            self.page.evaluate(f"navigator.clipboard.writeText({repr(source)})")
            self.page.keyboard.press("ControlOrMeta+v")

    def find_item(self, name: str) -> Locator | None:
        """Find an Arc item in the panel by name."""
        self._show_arc_panel()
        toolbar = self.page.locator(self.TOOLBAR_CLASS)
        items = toolbar.locator(self.LIST_ITEM_CLASS).filter(has_text=name)
        if items.count() == 0:
            return None
        return items.first

    def get_item(self, name: str) -> Locator:
        """Get an Arc item locator from the panel."""
        item = self.find_item(name)
        if item is None:
            raise ValueError(f"Arc '{name}' not found in panel")
        return item

    def open(self, name: str) -> None:
        """Open an Arc by double-clicking its item in the panel."""
        self._show_arc_panel()
        item = self.get_item(name)
        item.dblclick()
        self.page.locator("[data-mode-id='arc']").wait_for(
            state="visible", timeout=5000
        )

    def select_rack(self, rack_name: str) -> None:
        """Select a rack from the rack dropdown in the Arc editor controls."""
        controls = self._get_controls()
        rack_dropdown = controls.locator("button").first
        rack_dropdown.wait_for(state="visible", timeout=5000)
        self.console.notifications.close_all()
        rack_dropdown.click()
        self.console.select_from_dropdown(rack_name, placeholder="Search")

    def configure(self) -> None:
        """Click the Configure button in the Arc editor controls."""
        controls = self._get_controls()
        configure_btn = controls.locator("button:has-text('Configure')")
        configure_btn.wait_for(state="visible", timeout=5000)
        self.console.notifications.close_all()
        configure_btn.click()
        controls.locator("text=Task configured successfully").wait_for(
            state="visible", timeout=15000
        )

    def start(self) -> None:
        """Click the Play button to start the Arc."""
        controls = self._get_controls()
        play_btn = controls.locator("button:has(.pluto-icon--play)")
        play_btn.wait_for(state="visible", timeout=5000)
        self.console.notifications.close_all()
        play_btn.click()
        controls.locator("text=Task started successfully").wait_for(
            state="visible", timeout=15000
        )

    def stop(self) -> None:
        """Click the Pause button to stop the Arc."""
        self._show_arc_panel()
        controls = self._get_controls()
        pause_btn = controls.locator("button:has(.pluto-icon--pause)")
        pause_btn.wait_for(state="visible", timeout=5000)
        self.console.notifications.close_all()
        pause_btn.click()
        controls.locator("text=Task stopped successfully").wait_for(
            state="visible", timeout=15000
        )

    def is_running(self) -> bool:
        """Check if the Arc is currently running by looking for the pause button."""
        self._show_arc_panel()
        controls = self.page.locator(self.CONTROLS_CLASS)
        if controls.count() == 0:
            return False
        pause_btn = controls.locator("button:has(.pluto-icon--pause)")
        return pause_btn.count() > 0 and pause_btn.is_visible()

    def delete(self, name: str) -> None:
        """Delete an Arc via context menu."""
        self._show_arc_panel()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click(button="right")
        context_delete = self.page.get_by_text("Delete", exact=True)
        context_delete.wait_for(state="visible", timeout=3000)
        context_delete.click()
        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=3000)
        delete_btn.click()
