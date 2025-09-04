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
from typing import Any, Optional

from playwright.sync_api import sync_playwright

from framework.test_case import TestCase


class Playwright(TestCase):
    """
    Playwright TestCase implementation
    """

    def setup(self) -> None:

        # Do not run on Windows with webkit
        headless = self.params.get("headless", True)
        default_timeout = self.params.get("default_timeout", 1000)  # 1s
        default_nav_timeout = self.params.get("default_nav_timeout", 1000)  # 1s

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
        port = "9090"
        self.page.goto(f"http://localhost:{port}/", timeout=10000)
        if "Core built without embedded console" in self.page.content():
            port = "5173"
            self.page.goto(f"http://localhost:{port}/", timeout=5000)

        self._log_message(f"Console found on port {port}")

        # Wait for and fill login form
        self.page.wait_for_selector("input", timeout=10000)
        self.page.locator("input").first.fill("synnax")
        self.page.locator('input[type="password"]').fill("seldon")
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

        if page_name is not None:
            tab = self.page.locator("div").filter(
                has_text=re.compile(f"^{re.escape(page_type)}$")
            )
            tab.dblclick()
            self.page.get_by_text(page_type).first.fill(page_name)
            self.page.keyboard.press("Enter")  # Confirm the change
            return None
        else:
            return None

    def command_palette(self, command: str) -> None:
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        self.page.wait_for_selector(f"text={command}", timeout=5000)
        self.page.get_by_text(command).click()

    def determine_browser(self) -> Any:
        """
        Provide random coverage for all browsers.
        """

        browsers = ["chromium", "firefox", "webkit"]
        browsers = [
            "chromium"
        ]  # Failing on Firefox in CI only. Keep as single browser until debugged.

        selected = random.choice(browsers)
        self._log_message(f"Randomly selected browser: {selected}")
        return getattr(self.playwright, selected)

    @property
    def ESCAPE(self) -> None:
        self.page.keyboard.press("Escape")

    @property
    def ENTER(self) -> None:
        self.page.keyboard.press("Enter")
