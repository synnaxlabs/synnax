#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import platform
import random
from typing import TYPE_CHECKING, Any

import synnax as sy
from playwright.sync_api import Locator, Page

from framework.utils import get_results_path

if TYPE_CHECKING:
    from .console import Console
    from .log import Log
    from .plot import Plot


class WorkspaceClient:
    """Workspace management for Console UI automation."""

    def __init__(self, page: Page, console: "Console"):
        """Initialize the workspace client.

        Args:
            page: Playwright Page instance
            console: Console instance for UI interactions
        """
        self.page = page
        self.console = console

    def get_item(self, name: str) -> Locator:
        """Get a workspace item locator from the resources toolbar.

        Args:
            name: Name of the workspace

        Returns:
            Locator for the workspace item
        """
        return self.page.locator("div[id^='workspace:']").filter(has_text=name).first

    def exists(self, name: str) -> bool:
        """Check if a workspace exists in the resources toolbar.

        Args:
            name: Name of the workspace to check

        Returns:
            True if workspace exists, False otherwise
        """
        self.console.show_resource_toolbar("workspace")
        try:
            self.page.locator("div[id^='workspace:']").first.wait_for(
                state="visible", timeout=2000
            )
        except Exception:
            return False
        count = self.page.locator("div[id^='workspace:']").filter(has_text=name).count()
        return count > 0

    def expand_active(self) -> None:
        """Expand the active workspace in the resources toolbar to show its contents."""
        self.console.show_resource_toolbar("workspace")
        workspace_item = self.page.locator("div[id^='workspace:']").first
        workspace_item.wait_for(state="visible", timeout=10000)
        caret = workspace_item.locator(".pluto--location-bottom")
        if caret.count() > 0:
            return
        workspace_item.click()

    # SY-3576
    def refresh_tree(self) -> None:
        """Collapse and re-expand the workspace to refresh the tree contents."""
        self.console.show_resource_toolbar("workspace")
        workspace_item = self.page.locator("div[id^='workspace:']").first
        if workspace_item.count() == 0:
            return  # No workspace exists yet on fresh core run
        workspace_item.wait_for(state="visible", timeout=5000)
        caret = workspace_item.locator(".pluto--location-bottom")
        if caret.count() > 0:
            workspace_item.click()
            sy.sleep(0.1)
        workspace_item.click()
        sy.sleep(0.1)

    def get_page(self, name: str) -> Locator:
        """Get a page item locator from the workspace resources toolbar.

        Args:
            name: Name of the page (schematic, line plot, etc.)

        Returns:
            Locator for the page item
        """
        return self.page.locator(".pluto-tree__item").filter(has_text=name).first

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
        except Exception:
            return False

    def open_page(self, name: str) -> None:
        """Open a page by double-clicking it in the workspace resources toolbar.

        Args:
            name: Name of the page to open
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.dblclick()
        self.console.close_nav_drawer()

    def drag_page_to_mosaic(self, name: str) -> None:
        """Drag a page from the workspace resources toolbar onto the mosaic.

        Args:
            name: Name of the page to drag
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        mosaic = self.page.locator(".console-mosaic").first
        page_item.drag_to(mosaic)
        self.console.close_nav_drawer()

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
        self.page.get_by_text("Rename", exact=True).click(timeout=5000)
        self.console.select_all_and_type(new_name)
        self.console.ENTER
        self.refresh_tree()

    def delete_page(self, name: str) -> None:
        """Delete a page via context menu in the workspace resources toolbar.

        Args:
            name: Name of the page to delete
        """
        self.expand_active()
        page_item = self.get_page(name)
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")
        self.page.get_by_text("Delete", exact=True).click(timeout=5000)
        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.refresh_tree()
        self.console.close_nav_drawer()

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
        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        delete_item = menu.get_by_text("Delete", exact=True)
        ungroup_item = menu.get_by_text("Ungroup", exact=True)
        if delete_item.count() > 0:
            delete_item.click(timeout=5000)
        else:
            ungroup_item.click(timeout=5000)
        self.console.close_nav_drawer()

    def delete_pages(self, names: list[str]) -> None:
        """Delete multiple pages via multi-select and context menu.

        Args:
            names: List of page names to delete
        """
        if not names:
            return

        self.expand_active()

        modifier = "Meta" if platform.system() == "Darwin" else "Control"

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            self.page.keyboard.down(modifier)
            page_item.click()
            self.page.keyboard.up(modifier)

        last_item = self.get_page(names[-1])
        last_item.click(button="right")

        self.page.get_by_text("Delete", exact=True).click(timeout=5000)
        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.refresh_tree()
        self.console.close_nav_drawer()

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
        menu = self.page.locator(".pluto-menu-context")
        menu.wait_for(state="visible", timeout=2000)
        menu.get_by_text("Copy link", exact=True).click(timeout=5000)
        self.console.close_nav_drawer()

        link: str = str(self.page.evaluate("navigator.clipboard.readText()"))

        return link

    def group_pages(self, names: list[str], group_name: str) -> None:
        """Group multiple pages into a folder via multi-select and context menu.

        Args:
            names: List of page names to group
            group_name: Name for the new group/folder
        """
        if not names:
            return

        self.expand_active()

        modifier = "Meta" if platform.system() == "Darwin" else "Control"

        first_item = self.get_page(names[0])
        first_item.wait_for(state="visible", timeout=5000)
        first_item.click()

        for name in names[1:]:
            page_item = self.get_page(name)
            page_item.wait_for(state="visible", timeout=5000)
            self.page.keyboard.down(modifier)
            page_item.click()
            self.page.keyboard.up(modifier)

        last_item = self.get_page(names[-1])
        last_item.click(button="right")

        self.page.get_by_text("Group Selection", exact=True).click(timeout=5000)
        self.console.select_all_and_type(group_name)
        self.console.ENTER
        self.refresh_tree()

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
        page_item.wait_for(state="visible", timeout=5000)
        page_item.click(button="right")
        self.page.evaluate("delete window.showSaveFilePicker")

        with self.page.expect_download(timeout=5000) as download_info:
            self.page.get_by_text("Export", exact=True).first.click(timeout=5000)

        download = download_info.value
        save_path = get_results_path(f"{name}_export.json")
        download.save_as(save_path)
        self.console.close_nav_drawer()

        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

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
            self.console.command_palette("Create a Workspace")
        else:
            self.console.close_nav_drawer()
            selector = (
                self.page.locator("button.pluto-dialog__trigger")
                .filter(has=self.page.locator(".pluto-icon--workspace"))
                .first
            )
            selector.click(timeout=5000)
            self.page.get_by_role("button", name="New", exact=True).click(timeout=5000)

        name_input = self.page.locator("input[placeholder='Workspace Name']")
        name_input.wait_for(state="visible", timeout=5000)
        name_input.fill(name)
        self.page.get_by_role("button", name="Create", exact=True).click(timeout=5000)
        name_input.wait_for(state="hidden", timeout=5000)
        self.console.show_resource_toolbar("workspace")
        self.get_item(name).wait_for(state="visible", timeout=5000)
        self.refresh_tree()
        return True

    def select(self, name: str) -> None:
        """Select a workspace from the resources toolbar.

        Args:
            name: Name of the workspace to select
        """
        selector = (
            self.page.locator("button.pluto-dialog__trigger")
            .filter(has=self.page.locator(".pluto-icon--workspace"))
            .first
        )
        if name in selector.inner_text():
            return
        self.console.show_resource_toolbar("workspace")
        self.get_item(name).dblclick(timeout=5000)
        self.page.get_by_role("button").filter(has_text=name).wait_for(
            state="visible", timeout=5000
        )
        self.console.close_nav_drawer()

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename a workspace via context menu.

        Args:
            old_name: Current name of the workspace
            new_name: New name for the workspace
        """
        self.console.show_resource_toolbar("workspace")
        workspace = self.get_item(old_name)
        workspace.wait_for(state="visible", timeout=5000)
        workspace.click(button="right")
        self.page.get_by_text("Rename", exact=True).click(timeout=5000)
        self.console.select_all_and_type(new_name)
        self.console.ENTER
        self.refresh_tree()

    def delete(self, name: str) -> None:
        """Delete a workspace via context menu.

        Args:
            name: Name of the workspace to delete
        """
        self.console.show_resource_toolbar("workspace")

        workspace = self.get_item(name)
        workspace.wait_for(state="visible", timeout=5000)
        workspace.click(button="right", timeout=5000)

        self.page.get_by_text("Delete", exact=True).click(timeout=5000)

        delete_btn = self.page.get_by_role("button", name="Delete", exact=True)
        delete_btn.wait_for(state="visible", timeout=5000)
        delete_btn.click(timeout=5000)
        self.refresh_tree()

        self.console.close_nav_drawer()

    def ensure_selected(self, name: str) -> None:
        """Create a workspace if it doesn't exist and select it.

        Args:
            name: Name of the workspace to ensure is selected
        """
        self.create(name)
        self.select(name)

    def _create_plot_instance(self, client: sy.Synnax, page_name: str) -> "Plot":
        """Create a Plot instance after a line plot becomes visible.

        Args:
            client: Synnax client instance.
            page_name: The name of the plot page.

        Returns:
            Plot instance for the opened plot.
        """
        from .plot import Plot

        line_plot = self.page.locator(Plot.pluto_label)
        line_plot.first.wait_for(state="visible", timeout=5000)

        plot = Plot.__new__(Plot)
        plot.client = client
        plot.console = self.console
        plot.page = self.page
        plot.page_name = page_name
        plot.data = {"Y1": [], "Y2": [], "Ranges": [], "X1": None}
        plot.pane_locator = line_plot.first
        return plot

    def open_plot(self, client: sy.Synnax, name: str) -> "Plot":
        """Open a plot by double-clicking it in the workspace resources toolbar.

        Args:
            client: Synnax client instance.
            name: Name of the plot to open.

        Returns:
            Plot instance for the opened plot.
        """
        self.open_page(name)
        return self._create_plot_instance(client, name)

    def drag_plot_to_mosaic(self, client: sy.Synnax, name: str) -> "Plot":
        """Drag a plot from the workspace resources toolbar onto the mosaic.

        Args:
            client: Synnax client instance.
            name: Name of the plot to drag.

        Returns:
            Plot instance for the opened plot.
        """
        self.drag_page_to_mosaic(name)
        return self._create_plot_instance(client, name)

    def open_plot_from_search(self, client: sy.Synnax, name: str) -> "Plot":
        """Open a plot by searching its name in the command palette.

        Args:
            client: Synnax client instance.
            name: Name of the plot to search for and open.

        Returns:
            Plot instance for the opened plot.
        """
        self.console.search_palette(name)
        return self._create_plot_instance(client, name)

    def _create_log_instance(self, client: sy.Synnax, page_name: str) -> "Log":
        """Create a Log instance after a log becomes visible.

        Args:
            client: Synnax client instance.
            page_name: The name of the log page.

        Returns:
            Log instance for the opened log.
        """
        from .log import Log

        log_pane = self.page.locator(Log.pluto_label)
        log_pane.first.wait_for(state="visible", timeout=5000)

        log = Log.__new__(Log)
        log.client = client
        log.console = self.console
        log.page = self.page
        log.page_name = page_name
        log.pane_locator = log_pane.first
        return log

    def open_log(self, client: sy.Synnax, name: str) -> "Log":
        """Open a log by double-clicking it in the workspace resources toolbar.

        Args:
            client: Synnax client instance.
            name: Name of the log to open.

        Returns:
            Log instance for the opened log.
        """
        self.open_page(name)
        return self._create_log_instance(client, name)

    def drag_log_to_mosaic(self, client: sy.Synnax, name: str) -> "Log":
        """Drag a log from the workspace resources toolbar onto the mosaic.

        Args:
            client: Synnax client instance.
            name: Name of the log to drag.

        Returns:
            Log instance for the opened log.
        """
        self.drag_page_to_mosaic(name)
        return self._create_log_instance(client, name)

    def open_log_from_search(self, client: sy.Synnax, name: str) -> "Log":
        """Open a log by searching its name in the command palette.

        Args:
            client: Synnax client instance.
            name: Name of the log to search for and open.

        Returns:
            Log instance for the opened log.
        """
        self.console.search_palette(name)
        return self._create_log_instance(client, name)
