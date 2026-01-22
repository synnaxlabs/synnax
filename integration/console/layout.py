#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import re
from typing import TYPE_CHECKING

from playwright.sync_api import Locator, Page

if TYPE_CHECKING:
    from .console import Console


class LayoutClient:
    """Layout and tab management for Console UI automation."""

    def __init__(self, page: Page, console: "Console"):
        """Initialize the layout client.

        Args:
            page: Playwright Page instance
            console: Console instance for UI interactions
        """
        self.page = page
        self.console = console

    def get_tab(self, name: str) -> Locator:
        """Get a tab locator by its name.

        Args:
            name: The name/title of the tab to find

        Returns:
            Locator for the tab element
        """
        return (
            self.page.locator(".pluto-tabs-selector")
            .locator("div")
            .filter(has_text=re.compile(f"^{re.escape(name)}$"))
            .filter(has=self.page.locator("[aria-label='pluto-tabs__close']"))
            .first
        )

    def close_tab(self, name: str) -> None:
        """Close a tab using a randomly selected modality.

        Randomly chooses between:
        - Click close button (X)
        - Context menu -> Close

        Args:
            name: Name of the tab to close
        """
        self.console.close_nav_drawer()
        tab = self.get_tab(name)
        tab.wait_for(state="visible", timeout=5000)

        modality = random.choice(["button", "context_menu"])
        if modality == "button":
            tab.get_by_label("pluto-tabs__close").click()
        else:
            tab.locator("p").click(button="right")
            self.page.get_by_text("Close").first.click()

        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()

    def rename_tab(self, old_name: str, new_name: str) -> None:
        """Rename a tab using a randomly selected modality.

        Randomly chooses between:
        - Double-click on tab name
        - Context menu -> Rename

        Args:
            old_name: Current name of the tab
            new_name: New name for the tab
        """
        self.console.close_nav_drawer()
        tab = self.get_tab(old_name)
        tab.wait_for(state="visible", timeout=5000)

        modality = random.choice(["dblclick", "context_menu"])
        if modality == "dblclick":
            tab.locator("p").first.dblclick()
        else:
            tab.locator("p").click(button="right")
            self.page.get_by_text("Rename").first.click()

        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type(new_name)
        self.console.ENTER
        self.page.wait_for_timeout(200)
        self.get_tab(new_name).wait_for(state="visible", timeout=10000)

    def split_horizontal(self, tab_name: str) -> None:
        """Split a leaf horizontally via context menu.

        Args:
            tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        tab.click(button="right")
        self.page.get_by_text("Split Horizontally").first.click()

    def split_vertical(self, tab_name: str) -> None:
        """Split a leaf vertically via context menu.

        Args:
            tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        tab.click(button="right")
        self.page.get_by_text("Split Vertically").first.click()

    def focus(self, tab_name: str) -> None:
        """Focus on a leaf (maximize it) via context menu.

        Args:
            tab_name: Name of the tab to focus
        """
        tab = self.get_tab(tab_name)
        tab.click(button="right")
        self.page.get_by_text("Focus").first.click()

    def show_visualization_toolbar(self) -> None:
        """Show the visualization toolbar by pressing V."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        if bottom_drawer.count() == 0 or not bottom_drawer.is_visible():
            self.page.keyboard.press("V")
        bottom_drawer.wait_for(state="visible", timeout=5000)

    def hide_visualization_toolbar(self) -> None:
        """Hide the visualization toolbar by pressing V."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        if bottom_drawer.count() > 0 and bottom_drawer.is_visible():
            # Click on mosaic area to ensure focus before toggling
            self.page.locator(".console-mosaic").first.click()
            self.page.keyboard.press("V")
            bottom_drawer.wait_for(state="hidden", timeout=5000)

    def get_visualization_toolbar_title(self) -> str:
        """Get the title from the visualization toolbar header."""
        bottom_drawer = self.page.locator(
            ".console-nav__drawer.pluto--location-bottom.pluto--visible"
        )
        # Use combined selector to handle different page type structures
        header = bottom_drawer.locator(
            "header .pluto-breadcrumb__segment, header .pluto-header__text"
        ).first
        header.wait_for(state="visible", timeout=5000)
        return header.inner_text().strip()
