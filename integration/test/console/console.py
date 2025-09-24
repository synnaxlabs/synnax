#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import re
from typing import Optional

from playwright.sync_api import Locator, Page

from .channels import ChannelClient
from .console_page import ConsolePage
from .log import Log
from .plot import Plot
from .schematic import Schematic
from .table import Table


class Console:
    """
    Console UI automation interface with namespaced modules.
    Parallel to synnax client structure.
    """

    console_pages: list[ConsolePage]
    channels: ChannelClient

    def __init__(self, page: Page):

        # Playwright
        self.page = page
        self.channels = ChannelClient(page, self)
        self.schematic = Schematic(page, self)
        self.plot = Plot(page, self)
        self.log = Log(page, self)
        self.table = Table(page, self)

    def command_palette(self, command: str) -> None:
        """Execute a command via the command palette"""
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        self.page.wait_for_selector(f"text={command}", timeout=5000)
        self.page.get_by_text(command).click()

    @property
    def ESCAPE(self) -> None:
        self.page.keyboard.press("Escape")

    @property
    def ENTER(self) -> None:
        self.page.keyboard.press("Enter")

    def select_from_dropdown(self, input_field: str, input_text: str) -> None:
        """Helper method for dropdown selection"""
        channel_button = (
            self.page.locator(f"text={input_field}")
            .locator("..")
            .locator("button")
            .first
        )
        channel_button.click()
        self.select_from_dropdown_item(input_text, channel_button)

    def select_from_dropdown_item(self, text: str, dropdown_button_or_selector) -> None:
        """Select an item from an open dropdown."""
        self.page.wait_for_timeout(300)

        if isinstance(dropdown_button_or_selector, str):
            # Handle searchable dropdowns (string selector)
            search_input = self.page.locator(dropdown_button_or_selector)
            if search_input.count() > 0:
                search_input.fill(text)
                self.page.wait_for_timeout(300)
                dropdown_container = search_input.locator("..").locator("..")
                item_selector = dropdown_container.locator(".pluto-list__item").all()
        else:
            # Handle simple dropdowns (Locator) - try container first, fallback to global
            dropdown_container = dropdown_button_or_selector.locator("..")
            item_selector = dropdown_container.locator(".pluto-list__item").all()
            if len(item_selector) == 0:
                item_selector = self.page.locator(".pluto-list__item").all()

        item_found = False
        for item in item_selector:
            if item.is_visible() and text.lower() in item.inner_text().strip().lower():
                item.click()
                item_found = True
                break

        if not item_found:
            raise RuntimeError(f"Could not find item '{text}' in dropdown")

    def create_page(
        self, page_type: str, page_name: Optional[str] = None
    ) -> tuple[Locator, str]:
        """Create a new page via command palette"""
        # Handle "a" vs "an" article for proper command matching
        vowels = ["A", "E", "I", "O", "U"]
        # Special case for "NI" (en-eye)
        article = (
            "an"
            if page_type[0].upper() in vowels or page_type.startswith("NI")
            else "a"
        )
        page_command = f"Create {article} {page_type}"

        self.command_palette(page_command)

        # Wait for page to be created - use a simple timeout approach
        self.page.wait_for_timeout(1000)  # Give time for page creation

        # Try to find the newly created page/tab by page_type text
        # Look for the page type text which should appear after creation
        page_tab = (
            self.page.locator("div")
            .filter(has_text=re.compile(f"^{re.escape(page_type)}$"))
            .first
        )
        page_id = page_tab.inner_text().strip()

        # If page name provided, rename the page
        if page_name is not None:
            page_tab.dblclick()
            self.page.get_by_text(page_type).first.fill(page_name)
            self.page.keyboard.press("Enter")  # Confirm the change
            page_id = page_name  # Update page_id to the custom name

        return page_tab, page_id

    def close_page(self, page_name: str) -> None:
        """
        Close a page by name.
        Ignore unsaved changes.
        """
        tab = self.page.locator("div").filter(
            has_text=re.compile(f"^{re.escape(page_name)}$")
        )
        tab.get_by_label("pluto-tabs__close").click()

        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()

    def screenshot(self, path: Optional[str] = None) -> None:
        """Take a screenshot of the entire console page."""
        if path is None:
            os.makedirs("test/results", exist_ok=True)
            path = "test/results/console.png"

        self.page.screenshot(
            path=path, full_page=True, animations="disabled", type="png"
        )
