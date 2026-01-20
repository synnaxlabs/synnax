#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import synnax as sy
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

    def rename_tab(self, old_name: str, new_name: str) -> None:
        """Rename a tab by double-clicking and typing a new name.

        Args:
            old_name: Current name of the tab
            new_name: New name for the tab
        """
        self.console.close_nav_drawer()

        tab = self.get_tab(old_name)
        tab.wait_for(state="visible", timeout=5000)
        text_element = tab.locator("p").first
        text_element.dblclick()
        self.page.keyboard.press("ControlOrMeta+a")
        self.page.keyboard.type(new_name)
        self.page.keyboard.press("Enter")
        self.get_tab(new_name).wait_for(state="visible", timeout=5000)

    def split_horizontal(self, tab_name: str) -> None:
        """Split a leaf horizontally via context menu.

        Args:
            tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        tab.click(button="right")
        sy.sleep(0.3)
        self.page.get_by_text("Split Horizontally").first.click()
        sy.sleep(0.2)

    def split_vertical(self, tab_name: str) -> None:
        """Split a leaf vertically via context menu.

        Args:
            tab_name: Name of the tab to split
        """
        tab = self.get_tab(tab_name)
        tab.click(button="right")
        sy.sleep(0.3)
        self.page.get_by_text("Split Vertically").first.click()
        sy.sleep(0.2)

    def focus(self, tab_name: str) -> None:
        """Focus on a leaf (maximize it) via context menu.

        Args:
            tab_name: Name of the tab to focus
        """
        tab = self.get_tab(tab_name)
        tab.click(button="right")
        sy.sleep(0.3)
        self.page.get_by_text("Focus").first.click()
        sy.sleep(0.2)
