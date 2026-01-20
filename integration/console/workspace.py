#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
from typing import TYPE_CHECKING

import synnax as sy
from playwright.sync_api import Locator, Page

if TYPE_CHECKING:
    from .console import Console


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
                state="visible", timeout=300
            )
        except Exception:
            return False
        return (
            self.page.locator("div[id^='workspace:']").filter(has_text=name).count() > 0
        )

    def expand_active(self) -> None:
        """Expand the active workspace in the resources toolbar to show its contents."""
        self.console.show_resource_toolbar("workspace")
        workspace_item = self.page.locator("div[id^='workspace:']").first
        workspace_item.wait_for(state="visible", timeout=5000)
        caret = workspace_item.locator(".pluto--location-bottom")
        if caret.count() > 0:
            return
        workspace_item.click()

    # SY-3576
    def refresh_tree(self) -> None:
        """Collapse and re-expand the workspace to refresh the tree contents."""
        self.console.show_resource_toolbar("workspace")
        workspace_item = self.page.locator("div[id^='workspace:']").first
        workspace_item.wait_for(state="visible", timeout=5000)
        caret = workspace_item.locator(".pluto--location-bottom")
        if caret.count() > 0:
            workspace_item.click()
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

    def page_exists(self, name: str) -> bool:
        """Check if a page (schematic, line plot, etc.) exists in the workspace.

        Args:
            name: Name of the page to check

        Returns:
            True if page exists, False otherwise
        """
        self.expand_active()
        return self.get_page(name).count() > 0

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
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type(new_name)
        self.console.ENTER
        self.refresh_tree()

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
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type(new_name)
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
        self.console.close_nav_drawer()
