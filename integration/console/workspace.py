#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import random
from typing import TYPE_CHECKING, Any, Literal

import synnax as sy
from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from framework.utils import get_results_path

if TYPE_CHECKING:
    from .console import Console
    from .layout import LayoutClient
    from .log import Log
    from .plot import Plot
    from .schematic import Schematic


PageType = Literal[
    "Control Sequence",
    "Line Plot",
    "Schematic",
    "Log",
    "Table",
    "NI Analog Read Task",
    "NI Analog Write Task",
    "NI Counter Read Task",
    "NI Digital Read Task",
    "NI Digital Write Task",
    "LabJack Read Task",
    "LabJack Write Task",
    "OPC UA Read Task",
    "OPC UA Write Task",
]


class WorkspaceClient:
    """Workspace management for Console UI automation."""

    def __init__(self, layout: "LayoutClient", console: "Console"):
        self.layout = layout
        self.console = console

    def create_page(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Create a new page via New Page (+) button or command palette (randomly chosen)."""
        self.layout.close_nav_drawer()
        if random.random() < 0:
            return self._create_page_by_new_page_button(page_type, page_name)
        return self._create_page_by_command_palette(page_type, page_name)

    def _create_page_by_new_page_button(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Create a new page via the New Page (+) button."""
        self.layout.close_nav_drawer()
        add_btn = self.layout.page.locator(
            ".console-mosaic > .pluto-tabs-selector .pluto-tabs-selector__actions button:has(.pluto-icon--add)"
        ).first
        add_btn.wait_for(state="visible", timeout=5000)
        add_btn.click(force=True)

        self.layout.page.locator(".console-layout-selector__frame").wait_for(
            state="visible", timeout=15000
        )
        self.layout.page.get_by_role("button", name=page_type).first.click()

        return self._handle_new_page(page_type, page_name)

    def _create_page_by_command_palette(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Create a new page via command palette."""
        self.layout.close_nav_drawer()

        vowels = ["A", "E", "I", "O", "U"]
        article = (
            "an"
            if page_type[0].upper() in vowels or page_type.startswith("NI")
            else "a"
        )
        self.layout.command_palette(f"Create {article} {page_type}")
        return self._handle_new_page(page_type, page_name)

    def _handle_new_page(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Handle the new page creation after clicking create button."""
        sy.sleep(0.2)
        modal_was_open = self.layout.is_modal_open()
        tab_name: str = page_type

        if modal_was_open:
            tab_name = page_name if page_name is not None else page_type
            name_input = self.layout.page.get_by_role("textbox", name="Name")
            name_input.fill(tab_name)
            name_input.press("ControlOrMeta+Enter")

        page_tab = self.layout.get_tab(tab_name)
        page_tab.wait_for(state="visible", timeout=15000)
        page_id = page_tab.inner_text().strip()

        if page_name is not None and not modal_was_open:
            self.layout.rename_tab(old_name=tab_name, new_name=page_name)
            page_id = page_name
            page_tab = self.layout.get_tab(page_name)

        return page_tab, page_id

    def close_page(self, page_name: str) -> None:
        """Close a page by name. Ignores unsaved changes."""
        self.layout.close_tab(page_name)

    def get_item(self, name: str) -> Locator:
        """Get a workspace item locator from the resources toolbar.

        Args:
            name: Name of the workspace

        Returns:
            Locator for the workspace item
        """
        return self.layout.page.locator("div[id^='workspace:']").filter(has_text=name).first

    def exists(self, name: str) -> bool:
        """Check if a workspace exists in the resources toolbar.

        Args:
            name: Name of the workspace to check

        Returns:
            True if workspace exists, False otherwise
        """
        self.layout.show_resource_toolbar("workspace")
        try:
            self.layout.page.locator("div[id^='workspace:']").first.wait_for(
                state="visible", timeout=2000
            )
        except PlaywrightTimeoutError:
            return False
        count = self.layout.page.locator("div[id^='workspace:']").filter(has_text=name).count()
        return count > 0

    def wait_for_workspace_removed(self, name: str) -> None:
        """Wait for a workspace to be removed from the resources toolbar.

        Args:
            name: Name of the workspace to wait for removal
            timeout: Maximum time in milliseconds to wait
        """
        self.layout.show_resource_toolbar("workspace")
        workspace_item = self.layout.page.locator("div[id^='workspace:']").filter(
            has_text=name
        )
        workspace_item.first.wait_for(state="hidden", timeout=5000)

    def expand_active(self) -> None:
        """Expand the active workspace in the resources toolbar to show its contents."""
        self.layout.show_resource_toolbar("workspace")
        workspace_item = self.layout.page.locator("div[id^='workspace:']").first
        workspace_item.wait_for(state="visible", timeout=10000)
        caret = workspace_item.locator(".pluto--location-bottom")
        if caret.count() > 0:
            return
        workspace_item.click()
        caret.wait_for(state="visible", timeout=5000)

    def get_page(self, name: str) -> Locator:
        """Get a page item locator from the workspace resources toolbar.

        Args:
            name: Name of the page (schematic, line plot, etc.)

        Returns:
            Locator for the page item
        """
        return self.layout.page.locator(".pluto-tree__item").filter(has_text=name).first

    def page_exists(self, name: str, timeout: int = 2000) -> bool:
        """Check if a page (schematic, line plot, etc.) exists in the workspace.

        Args:
            name: Name of the page to check
            timeout: Maximum time in milliseconds to wait for the page to appear

        Returns:
            True if page exists, False otherwise
        """
        self.expand_active()
        try:
            self.get_page(name).wait_for(state="visible", timeout=timeout)
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_for_page_removed(self, name: str, timeout: int = 5000) -> None:
        """Wait for a page to be removed from the workspace.

        Args:
            name: Name of the page to wait for removal
            timeout: Maximum time in milliseconds to wait
        """
        page_item = self.get_page(name)
        page_item.wait_for(state="hidden", timeout=timeout)

    def open_page(self, name: str) -> None:
        """Open a page by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the page to open
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.dblclick()
        self.layout.close_nav_drawer()

    def drag_page_to_mosaic(self, name: str) -> None:
        """Drag a page from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the page to drag
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        mosaic = self.layout.page.locator(".console-mosaic").first
        page_item.drag_to(mosaic)
        self.layout.close_nav_drawer()

    def rename_page(self, old_name: str, new_name: str) -> None:
        """Rename a page via context menu in the workspace resources toolbar.

        Args:
            old_name: Current name of the page
            new_name: New name for the page
        """
        self.expand_active()
        page_item = self.get_page(old_name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")
        self.layout.page.get_by_text("Rename", exact=True).click(timeout=5000)
        self.layout.select_all_and_type(new_name)
        self.layout.page.keyboard.press("Enter")
        self.get_page(new_name).wait_for(state="visible", timeout=5000)
        self.wait_for_page_removed(old_name)
        self.layout.close_nav_drawer()

    def delete_page(self, name: str) -> None:
        """Delete a page via context menu in the workspace resources toolbar.

        Args:
            name: Name of the page to delete
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")
        self.layout.page.get_by_text("Delete", exact=True).click(timeout=5000)
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.wait_for_page_removed(name)
        self.layout.close_nav_drawer()

    def delete_group(self, name: str) -> None:
        """Delete a group via context menu.

        Groups are deleted immediately without a confirmation dialog (unlike pages).
        The context menu shows "Delete" for collapsed groups and "Ungroup" for expanded
        groups with visible children.

        Args:
            name: Name of the group to delete
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")
        menu = self.layout.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        delete_item = menu.get_by_text("Delete", exact=True)
        ungroup_item = menu.get_by_text("Ungroup", exact=True)
        if delete_item.count() > 0:
            delete_item.click(timeout=5000)
        else:
            ungroup_item.click(timeout=5000)
        self.layout.close_nav_drawer()

    def delete_pages(self, names: list[str]) -> None:
        """Delete multiple pages via multi-select and context menu.

        Args:
            names: List of page names to delete
        """
        if not names:
            return

        self.expand_active()

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        last_item.click(button="right")

        self.layout.page.get_by_text("Delete", exact=True).click(timeout=5000)
        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        for name in names:
            self.wait_for_page_removed(name)
        self.layout.close_nav_drawer()

    def copy_page_link(self, name: str) -> str:
        """Copy link to a page via context menu.

        Args:
            name: Name of the page to copy link for

        Returns:
            The copied link from clipboard
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")
        menu = self.layout.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        menu.get_by_text("Copy link", exact=True).click(timeout=5000)
        self.layout.close_nav_drawer()

        link: str = str(self.layout.page.evaluate("navigator.clipboard.readText()"))

        return link

    def group_pages(self, *, names: list[str], group_name: str) -> None:
        """Group multiple pages into a folder via multi-select and context menu.

        Args:
            names: List of page names to group
            group_name: Name for the new group/folder
        """
        if not names:
            return

        self.expand_active()

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        last_item.click(button="right")

        self.layout.page.get_by_text("Group Selection", exact=True).click(timeout=5000)
        self.layout.select_all_and_type(group_name)
        self.layout.page.keyboard.press("Enter")
        self.layout.close_nav_drawer()

    def export_page(self, name: str) -> dict[str, Any]:
        """Export a page via context menu.

        The file is saved to the tests/results directory.

        Args:
            name: Name of the page to export

        Returns:
            The exported JSON content as a dictionary
        """
        self.expand_active()
        page_item = self.get_page(name)
        try:
            page_item.wait_for(state="visible", timeout=5000)
        except Exception as e:
            all_items = self.layout.page.locator(".pluto-tree__item").all()
            item_texts = [
                item.text_content() for item in all_items if item.is_visible()
            ]
            raise Exception(
                f"Page '{name}' not found. Available items: {item_texts}"
            ) from e
        page_item.click(button="right")
        self.layout.page.evaluate("delete window.showSaveFilePicker")

        with self.layout.page.expect_download(timeout=5000) as download_info:
            self.layout.page.get_by_text("Export", exact=True).first.click(timeout=5000)

        download = download_info.value
        save_path = get_results_path(f"{name}_export.json")
        download.save_as(save_path)
        self.layout.close_nav_drawer()

        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def snapshot_page_to_active_range(self, name: str, range_name: str) -> None:
        """Snapshot a page to the active range via context menu.

        Args:
            name: Name of the page to snapshot
            range_name: Name of the active range (for menu text matching)
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")

        snapshot_item = self.layout.page.get_by_text(f"Snapshot to {range_name}", exact=True)
        snapshot_item.wait_for(state="visible", timeout=5000)
        snapshot_item.click(timeout=5000)
        self.layout.close_nav_drawer()

    def snapshot_pages_to_active_range(self, names: list[str], range_name: str) -> None:
        """Snapshot multiple pages to the active range via context menu.

        Args:
            names: List of page names to snapshot
            range_name: Name of the active range (for menu text matching)
        """
        if not names:
            return

        self.expand_active()

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        last_item.click(button="right")

        snapshot_item = self.layout.page.get_by_text(f"Snapshot to {range_name}", exact=True)
        snapshot_item.wait_for(state="visible", timeout=5000)
        snapshot_item.click(timeout=5000)
        self.layout.close_nav_drawer()

    def copy_page(self, name: str, new_name: str) -> None:
        """Make a copy of a page via context menu.

        Args:
            name: Name of the page to copy
            new_name: Name for the new copy
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")

        copy_item = self.layout.page.get_by_text("Copy", exact=True)
        copy_item.wait_for(state="visible", timeout=5000)
        copy_item.click(timeout=5000)

        self.layout.select_all_and_type(new_name)
        self.layout.page.keyboard.press("Enter")
        self.get_page(new_name).wait_for(state="visible", timeout=5000)
        self.layout.close_nav_drawer()

    def copy_pages(self, names: list[str]) -> None:
        """Copy multiple pages via context menu.

        Note: When copying multiple pages, each gets a " (copy)" suffix automatically.

        Args:
            names: List of page names to copy
        """
        if not names:
            return

        self.expand_active()

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            page_item.click(modifiers=["ControlOrMeta"])

        # For single item, reuse first_item; otherwise get the last item
        last_item = first_item if len(names) == 1 else self.get_page(names[-1])
        last_item.click(button="right")

        copy_item = self.layout.page.get_by_text("Copy", exact=True)
        copy_item.wait_for(state="visible", timeout=5000)
        copy_item.click(timeout=5000)

        for name in names:
            copy_name = f"{name} (copy)"
            self.get_page(copy_name).wait_for(state="visible", timeout=5000)

        self.layout.close_nav_drawer()

    def create(self, name: str) -> bool:
        """Create a workspace via command palette.

        Args:
            name: Name of the workspace to create

        Returns:
            True if workspace was created, False if it already exists
        """
        if self.exists(name):
            return False

        if random.choice([True, False]):
            self.layout.command_palette("Create a Workspace")
        else:
            self.layout.close_nav_drawer()
            selector = (
                self.layout.page.locator("button.pluto-dialog__trigger")
                .filter(has=self.layout.page.locator(".pluto-icon--workspace"))
                .first
            )
            selector.click(timeout=5000)
            self.layout.page.get_by_role("button", name="New", exact=True).click(timeout=5000)

        name_input = self.layout.page.locator("input[placeholder='Workspace Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.layout.page.get_by_role("button", name="Create", exact=True).click(timeout=5000)
        name_input.wait_for(state="hidden", timeout=5000)
        self.layout.show_resource_toolbar("workspace")
        self.get_item(name).wait_for(state="visible", timeout=5000)
        self.layout.close_nav_drawer()
        return True

    def select(self, name: str) -> None:
        """Select a workspace from the resources toolbar.

        Args:
            name: Name of the workspace to select
        """
        selector = (
            self.layout.page.locator("button.pluto-dialog__trigger")
            .filter(has=self.layout.page.locator(".pluto-icon--workspace"))
            .first
        )
        if name in selector.inner_text():
            return
        self.layout.show_resource_toolbar("workspace")
        self.get_item(name).dblclick(timeout=5000)
        self.layout.page.get_by_role("button").filter(has_text=name).wait_for(
            state="visible", timeout=5000
        )
        self.layout.close_nav_drawer()

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename a workspace via context menu.

        Args:
            old_name: Current name of the workspace
            new_name: New name for the workspace
        """
        self.layout.show_resource_toolbar("workspace")
        workspace = self.get_item(old_name)
        workspace.wait_for(state="visible", timeout=5000)
        workspace.click(button="right")
        self.layout.page.get_by_text("Rename", exact=True).click(timeout=5000)
        self.layout.select_all_and_type(new_name)
        self.layout.page.keyboard.press("Enter")
        self.layout.close_nav_drawer()

    def delete(self, name: str) -> None:
        """Delete a workspace via context menu.

        Args:
            name: Name of the workspace to delete
        """
        self.layout.show_resource_toolbar("workspace")

        workspace = self.get_item(name)
        workspace.wait_for(state="visible", timeout=5000)
        workspace.click(button="right", timeout=5000)

        self.layout.page.get_by_text("Delete", exact=True).click(timeout=5000)

        delete_btn = self.layout.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.wait_for_workspace_removed(name)
        self.layout.close_nav_drawer()

    def ensure_selected(self, name: str) -> None:
        """Create a workspace if it doesn't exist and select it.

        Args:
            name: Name of the workspace to ensure is selected
        """
        selector = self.layout.page.locator("button.pluto-dialog__trigger").filter(
            has=self.layout.page.locator(".pluto-icon--workspace")
        )
        if name in selector.inner_text(timeout=1000):
            return

        self.create(name)
        self.select(name)

    def open_plot(self, name: str) -> "Plot":
        """Open a plot by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the plot to open.

        Returns:
            Plot instance for the opened plot.
        """
        from .plot import Plot

        self.open_page(name)
        return Plot.from_open_page(self.console, name)

    def drag_plot_to_mosaic(self, name: str) -> "Plot":
        """Drag a plot from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the plot to drag.

        Returns:
            Plot instance for the opened plot.
        """
        from .plot import Plot

        self.drag_page_to_mosaic(name)
        return Plot.from_open_page(self.console, name)

    def open_log(self, name: str) -> "Log":
        """Open a log by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the log to open.

        Returns:
            Log instance for the opened log.
        """
        from .log import Log

        self.open_page(name)
        return Log.from_open_page(self.console, name)

    def drag_log_to_mosaic(self, name: str) -> "Log":
        """Drag a log from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the log to drag.

        Returns:
            Log instance for the opened log.
        """
        from .log import Log

        self.drag_page_to_mosaic(name)
        return Log.from_open_page(self.console, name)

    def open_schematic(self, name: str) -> "Schematic":
        """Open a schematic by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the schematic to open.

        Returns:
            Schematic instance for the opened schematic.
        """
        from .schematic import Schematic

        self.open_page(name)
        return Schematic.from_open_page(self.console, name)

    def drag_schematic_to_mosaic(self, name: str) -> "Schematic":
        """Drag a schematic from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the schematic to drag.

        Returns:
            Schematic instance for the opened schematic.
        """
        from .schematic import Schematic

        self.drag_page_to_mosaic(name)
        return Schematic.from_open_page(self.console, name)
