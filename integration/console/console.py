#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import random
import re
import time
from typing import Literal, Optional

from playwright.sync_api import Locator, Page

from .channels import ChannelClient
from .log import Log
from .page import ConsolePage
from .plot import Plot
from .schematic import Schematic

# Define literal types for page creation
PageType = Literal[
    "Control Sequence",
    "Line Plot",
    "Schematic",
    "Log",
    "Table",
    "NI Analog Read Task",
    "NI Analog Write Task",
    "NI Digital Read Task",
    "NI Digital Write Task",
    "LabJack Read Task",
    "LabJack Write Task",
    "OPC UA Read Task",
    "OPC UA Write Task",
]


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

    @property
    def MODAL_OPEN(self) -> bool:
        return (
            self.page.locator(
                "div.pluto-dialog__dialog.pluto--modal.pluto--visible"
            ).count()
            > 0
        )

    def select_from_dropdown(
        self, text: str, placeholder: Optional[str] = None
    ) -> None:
        """Select an item from an open dropdown."""
        self.page.wait_for_timeout(300)
        target_item = f".pluto-list__item:not(.pluto-tree__item):has-text('{text}')"

        if placeholder is not None:
            search_input = self.page.locator(f"input[placeholder*='{placeholder}']")
            if search_input.count() > 0:
                search_input.fill(text)
                self.page.wait_for_timeout(200)

        for attempt in range(10):
            try:
                self.page.wait_for_selector(target_item, timeout=100)
                self.page.locator(target_item).first.click()
                self.page.wait_for_timeout(100)
                return
            except Exception:
                self.page.wait_for_timeout(100)
                continue

        items = self.page.locator(
            ".pluto-list__item:not(.pluto-tree__item)"
        ).all_text_contents()
        raise RuntimeError(
            f"Could not find item '{text}' in dropdown. Available items: {items}"
        )

    def create_page(
        self, page_type: PageType, page_name: Optional[str] = None
    ) -> tuple[Locator, str]:
        """
        Public method for creating a new page in one of two ways:
        - By the New Page (+) button
        - By the command palette
        """

        if random.random() < 0.5:
            page_tab, page_id = self._create_page_by_new_page_button(
                page_type, page_name
            )
        else:
            page_tab, page_id = self._create_page_by_command_palette(
                page_type, page_name
            )

        return page_tab, page_id

    def _create_page_by_new_page_button(
        self, page_type: PageType, page_name: Optional[str] = None
    ) -> tuple[Locator, str]:
        """Create a new page via the New Page (+) button."""

        self.page.locator(".pluto-icon--add").first.click()  # (+)
        self.page.get_by_role("button", name=page_type).first.click()
        page_tab, page_id = self._handle_new_page(page_type, page_name)

        return page_tab, page_id

    def _create_page_by_command_palette(
        self, page_type: PageType, page_name: Optional[str] = None
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
        self.page.wait_for_timeout(100)
        page_tab, page_id = self._handle_new_page(page_type, page_name)

        return page_tab, page_id

    def _handle_new_page(
        self, page_type: PageType, page_name: Optional[str] = None
    ) -> tuple[Locator, str]:
        """Handle the new page creation"""
        if self.MODAL_OPEN:
            page_name = page_name or page_type
            self.page.get_by_role("textbox", name="Name").fill(page_name)
            self.page.get_by_role("textbox", name="Name").press("ControlOrMeta+Enter")

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
        tab = (
            self.page.locator("div")
            .filter(has_text=re.compile(f"^{re.escape(page_name)}$"))
            .first
        )
        tab.get_by_label("pluto-tabs__close").click()

        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()

    def screenshot(self, name: Optional[str] = None) -> None:
        """Take a screenshot of the entire console page."""
        results_dir = os.path.join(os.path.dirname(__file__), "..", "tests", "results")
        os.makedirs(results_dir, exist_ok=True)
        if name is None:
            name = "console.png"
        else:
            if not name.endswith(".png"):
                name = name + ".png"

        path = os.path.join(results_dir, name)
        self.page.screenshot(
            path=path, full_page=True, animations="disabled", type="png"
        )

    def click_btn(self, button_label: str) -> None:
        """Click a button by label."""
        button = (
            self.page.locator(f"text={button_label}")
            .locator("..")
            .locator("button")
            .first
        )
        button.click(force=True)

    def click_checkbox(self, checkbox_label: str) -> None:
        """Click a checkbox by label."""
        checkbox = (
            self.page.locator(f"text={checkbox_label}")
            .locator("..")
            .locator("input[type='checkbox']")
            .first
        )
        checkbox.click()

    def fill_input_field(self, input_label: str, value: str) -> None:
        """Fill an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        input_field.fill(value)

    def get_input_field(self, input_label: str) -> str:
        """Get the value of an input field by label."""
        input_field = (
            self.page.locator(f"text={input_label}")
            .locator("..")
            .locator("input")
            .first
        )
        return input_field.input_value()
