#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random
import re
import time
from test.framework.test_case import TestCase
from typing import Optional, cast

import synnax as sy
from playwright.sync_api import Browser, BrowserType, Page, sync_playwright


class Console(TestCase):
    """
    Console TestCase implementation using Playwright

    SY-2965: Break this out to Console & ConsoleCase
    """

    browser: Browser
    page: Page
    headless: bool
    default_timeout: int
    default_nav_timeout: int

    def setup(self) -> None:

        headless = self.params.get("headless", True)
        default_timeout = self.params.get("default_timeout", 5000)  # 5s
        default_nav_timeout = self.params.get("default_nav_timeout", 5000)  # 5s

        # Open page
        self._log_message(
            f"Opening browser in {'headless' if headless else 'visible'} mode"
        )
        self.playwright = sync_playwright().start()
        browser_engine = self.determine_browser()
        self.browser = browser_engine.launch(headless=headless)
        self.page = self.browser.new_page()

        # Set timeouts
        self.page.set_default_timeout(default_timeout)  # 1s
        self.page.set_default_navigation_timeout(default_nav_timeout)  # 1s

        # Try embedded console first, fallback to dev server if no console found
        host = self.synnax_connection.server_address
        port = self.synnax_connection.port

        self.page.goto(f"http://{host}:{port}/", timeout=10000)
        if "Core built without embedded console" in self.page.content():
            port = 5173
            self.page.goto(f"http://{host}:{port}/", timeout=5000)

        self._log_message(f"Console found on port {port}")

        # Wait for and fill login form
        username = self.synnax_connection.username
        password = self.synnax_connection.password
        self.page.wait_for_selector("input", timeout=10000)
        self.page.locator("input").first.fill(f"{username}")
        self.page.locator('input[type="password"]').fill(f"{password}")
        self.page.get_by_role("button", name="Sign In").click()
        self.page.wait_for_load_state("networkidle")

        # Toggle theme
        time.sleep(0.5)
        self.command_palette("Toggle Color Theme")

    def teardown(self) -> None:
        self.browser.close()

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

    def create_page(self, page_type: str, page_name: Optional[str] = None) -> None:
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

        if page_name is None:
            return None

        tab = self.page.locator("div").filter(
            has_text=re.compile(f"^{re.escape(page_type)}$")
        )
        tab.dblclick()
        self.page.get_by_text(page_type).first.fill(page_name)
        self.page.keyboard.press("Enter")  # Confirm the change

    def command_palette(self, command: str) -> None:
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        self.page.wait_for_selector(f"text={command}", timeout=5000)
        self.page.get_by_text(command).click()

    def determine_browser(self) -> BrowserType:
        """
        Provide random coverage for all browsers.
        """

        browsers = ["chromium", "firefox", "webkit"]
        # SY-2928
        # Firefox failing in CI only
        # Webkit failing on Win in CI
        browsers = ["chromium"]
        selected = random.choice(browsers)
        self._log_message(f"Randomly selected browser: {selected}")
        browser_attr = getattr(self.playwright, selected)
        return cast(BrowserType, browser_attr)

    @property
    def ESCAPE(self) -> None:
        self.page.keyboard.press("Escape")

    @property
    def ENTER(self) -> None:
        self.page.keyboard.press("Enter")

    def create_a_channel(
        self,
        channel_name: str,
        virtual: bool = False,
        is_index: bool = False,
        data_type: sy.CrudeDataType = sy.DataType.TIMESTAMP,
        index: str = "",
    ) -> bool:

        try:
            self.client.channels.retrieve(channel_name)
            self._log_message(f'Channel "{channel_name}" already exists')
            return False
        except sy.NotFoundError:
            self._log_message(f"Creating channel: {channel_name}")

        if is_index == False and index == "":
            raise ValueError("Index must be provided if is_index is False")

        self.command_palette("Create a Channel")

        name_input = self.page.locator("text=Name").locator("..").locator("input").first
        name_input.fill(channel_name)
        if virtual:
            self.page.get_by_text("Virtual").click()
        if is_index:
            is_index_toggle = (
                self.page.locator("text=Is Index")
                .locator("..")
                .locator("input[type='checkbox']")
                .first
            )
            is_index_toggle.click()
        else:
            data_type_str = str(sy.DataType(data_type))
            self._select_from_dropdown("Data Type", data_type_str)
            self._select_from_dropdown("Index", index)

        self.page.get_by_role("button", name="Create").click()
        self._log_message(f"Created channel {channel_name}")

        return True

    def _select_from_dropdown(self, input_field: str, input_text: str) -> None:

        channel_button = (
            self.page.locator(f"text={input_field}")
            .locator("..")
            .locator("button")
            .first
        )
        channel_button.click()
        search_input = self.page.locator("input[placeholder*='Search']")
        search_input.press("Control+a")
        search_input.type(input_text)
        self.page.wait_for_timeout(300)

        # Iterate through dropdown items
        item_found = False
        item_selector = self.page.locator(".pluto-list__item").all()
        for item in item_selector:
            if item.is_visible() and input_text in item.inner_text().strip().lower():
                item.click()
                item_found = True
                break

        if not item_found:
            raise RuntimeError(f"Could not find channel '{input_text}' in dropdown")
