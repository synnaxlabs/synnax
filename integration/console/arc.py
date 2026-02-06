#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator

from console.context_menu import ContextMenu
from console.layout import LayoutClient
from console.notifications import NotificationsClient
from console.tree import Tree


class ArcClient:
    """Arc automation management for Console UI automation."""

    ICON_NAME = "arc"
    TOOLBAR_CLASS = ".console-arc-toolbar"
    CONTROLS_CLASS = ".console-arc-editor__controls"
    LIST_ITEM_CLASS = ".pluto-list__item"

    def __init__(self, layout: LayoutClient):
        self.layout = layout
        self.ctx_menu = ContextMenu(layout.page)
        self.notifications = NotificationsClient(layout.page)
        self.tree = Tree(layout.page)

    def _show_arc_panel(self) -> None:
        """Show the Arc panel in the navigation drawer.

        Closes any open modal/dropdown first, then clicks the Arc button
        in the sidebar if the toolbar is not already visible.
        """
        self.layout.press_escape()
        toolbar = self.layout.page.locator(self.TOOLBAR_CLASS)
        if toolbar.count() > 0 and toolbar.is_visible():
            return
        arc_btn = self.layout.page.locator(f"button:has(.pluto-icon--{self.ICON_NAME})")
        arc_btn.click()
        toolbar.wait_for(state="visible", timeout=5000)

    def _get_controls(self) -> Locator:
        """Get the Arc editor controls locator.

        Returns:
            Locator for the Arc editor controls section.
        """
        controls = self.layout.page.locator(self.CONTROLS_CLASS)
        controls.wait_for(state="visible", timeout=5000)
        return controls

    def create(self, name: str, source: str, mode: str = "Text") -> None:
        """Create a new Arc automation via Console UI.

        Args:
            name: The name for the new Arc automation.
            source: The source code/content for the Arc.
            mode: The Arc mode, either "Text" or "Diagram". Defaults to "Text".
        """
        self._show_arc_panel()

        add_btn = self.layout.page.locator(f"{self.TOOLBAR_CLASS} button").first
        add_btn.click()

        name_input = self.layout.page.locator("input[placeholder='Automation Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)

        mode_btn = self.layout.page.locator(
            f".console-arc-create-modal__mode-select-button:has-text('{mode}')"
        )
        mode_btn.wait_for(state="visible", timeout=5000)
        mode_btn.click()

        create_btn = self.layout.page.get_by_role("button", name="Create", exact=True)
        create_btn.wait_for(state="visible", timeout=5000)
        create_btn.click()
        name_input.wait_for(state="hidden", timeout=10000)

        if mode == "Text":
            editor = self.layout.page.locator("[data-mode-id='arc']")
            editor.wait_for(state="visible", timeout=10000)
            editor.click()
            editor.locator(".cursor").wait_for(state="visible", timeout=5000)
            editor.locator(".monaco-editor.focused").wait_for(
                state="visible", timeout=5000
            )
            self.layout.press_key("ControlOrMeta+a")
            self.layout.page.evaluate(f"navigator.clipboard.writeText({repr(source)})")
            self.layout.press_key("ControlOrMeta+v")

    def find_item(self, name: str) -> Locator | None:
        """Find an Arc item in the panel by name.

        Args:
            name: The name of the Arc to find.

        Returns:
            Locator for the Arc item, or None if not found.
        """
        self._show_arc_panel()
        toolbar = self.layout.page.locator(self.TOOLBAR_CLASS)
        items = toolbar.locator(self.LIST_ITEM_CLASS).filter(has_text=name)
        if items.count() == 0:
            return None
        return items.first

    def get_item(self, name: str) -> Locator:
        """Get an Arc item locator from the panel.

        Args:
            name: The name of the Arc to get.

        Returns:
            Locator for the Arc item.

        Raises:
            ValueError: If the Arc is not found in the panel.
        """
        item = self.find_item(name)
        if item is None:
            raise ValueError(f"Arc '{name}' not found in panel")
        return item

    def open(self, name: str) -> None:
        """Open an Arc by double-clicking its item in the panel.

        Args:
            name: The name of the Arc to open.
        """
        self._show_arc_panel()
        item = self.get_item(name)
        item.dblclick()
        self.layout.page.locator("[data-mode-id='arc']").wait_for(
            state="visible", timeout=5000
        )

    def select_rack(self, rack_name: str) -> None:
        """Select a rack from the rack dropdown in the Arc editor controls.

        Args:
            rack_name: The name of the rack to select.
        """
        controls = self._get_controls()
        rack_dropdown = controls.locator("button").first
        rack_dropdown.wait_for(state="visible", timeout=5000)
        self.notifications.close_all()
        rack_dropdown.click()
        self.layout.select_from_dropdown(rack_name, placeholder="Search")

    def configure(self) -> None:
        """Click the Configure button in the Arc editor controls.

        Waits for the "Task configured successfully" message to appear.
        """
        controls = self._get_controls()
        configure_btn = controls.locator("button:has-text('Configure')")
        configure_btn.wait_for(state="visible", timeout=5000)
        self.notifications.close_all()
        configure_btn.click()
        controls.locator("text=Task configured successfully").wait_for(
            state="visible", timeout=15000
        )

    def start(self) -> None:
        """Click the Play button to start the Arc.

        Waits for the "Task started successfully" message to appear.
        """
        controls = self._get_controls()
        play_btn = controls.locator("button:has(.pluto-icon--play)")
        play_btn.wait_for(state="visible", timeout=5000)
        self.notifications.close_all()
        play_btn.click()
        controls.locator("text=Task started successfully").wait_for(
            state="visible", timeout=15000
        )

    def stop(self) -> None:
        """Click the Pause button to stop the Arc.

        Waits for the "Task stopped successfully" message to appear.
        """
        self._show_arc_panel()
        controls = self._get_controls()
        pause_btn = controls.locator("button:has(.pluto-icon--pause)")
        pause_btn.wait_for(state="visible", timeout=5000)
        self.notifications.close_all()
        pause_btn.click()
        controls.locator("text=Task stopped successfully").wait_for(
            state="visible", timeout=15000
        )

    def is_running(self) -> bool:
        """Check if the Arc is currently running by looking for the pause button.

        Returns:
            True if the Arc is running (pause button visible), False otherwise.
        """
        self._show_arc_panel()
        controls = self.layout.page.locator(self.CONTROLS_CLASS)
        if controls.count() == 0:
            return False
        pause_btn = controls.locator("button:has(.pluto-icon--pause)")
        return pause_btn.count() > 0 and pause_btn.is_visible()

    def delete(self, name: str) -> None:
        """Delete an Arc via context menu.

        Args:
            name: The name of the Arc to delete.
        """
        self._show_arc_panel()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.layout.delete_with_confirmation(item)
