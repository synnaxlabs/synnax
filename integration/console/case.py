#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import random
from typing import cast

from playwright.sync_api import Browser, BrowserType, Page, sync_playwright

from console.console import Console
from framework.test_case import TestCase


class ConsoleCase(TestCase):
    """
    Console TestCase implementation using Playwright

    Environment Variables:
    - PLAYWRIGHT_CONSOLE_HEADED: Run in headed mode (default: False)
      Can be set via command line: --console-headed or -ch
    """

    browser: Browser
    page: Page
    headed: bool
    default_timeout: int
    default_nav_timeout: int
    console: Console

    def setup(self) -> None:
        env_headed = os.environ.get("PLAYWRIGHT_CONSOLE_HEADED", "0") == "1"
        headed = self.params.get("headed", env_headed)
        slow_mo = self.params.get("slow_mo", 0)
        default_timeout = self.params.get("default_timeout", 15000)  # 15s
        default_nav_timeout = self.params.get("default_nav_timeout", 15000)  # 15s

        # Open page
        self.log(f"Opening browser in {'headed' if headed else 'headless'} mode")
        self.playwright = sync_playwright().start()
        browser_engine = self.determine_browser()
        self.browser = browser_engine.launch(headless=not headed, slow_mo=slow_mo)
        # Use larger viewport to reduce element overlap
        self.page = self.browser.new_page()

        # Set timeouts
        self.page.set_default_timeout(default_timeout)  # 1s
        self.page.set_default_navigation_timeout(default_nav_timeout)  # 1s

        # Try embedded console first, fallback to dev server if no console found
        host = self.synnax_connection.server_address
        port = self.synnax_connection.port

        self.page.goto(f"http://{host}:{port}/", timeout=20000)
        if "Core built without embedded console" in self.page.content():
            port = 5173
            self.page.goto(f"http://{host}:{port}/", timeout=15000)

        self.log(f"Console found on port {port}")

        # Wait for and fill login form
        username = self.synnax_connection.username
        password = self.synnax_connection.password

        self.page.wait_for_selector(".pluto-field__username", timeout=5000)
        username_input = self.page.locator(".pluto-field__username input").first
        username_input.fill(username)

        password_input = self.page.locator(".pluto-field__password input").first
        password_input.fill(password)

        login_button = self.page.get_by_role("button", name="Log In")
        login_button.wait_for(state="attached", timeout=2000)
        login_button.click()

        self.page.wait_for_load_state("networkidle")

        # Initialize Console interface
        self.console = Console(self.page)
        self.page.wait_for_selector("text=Get Started", timeout=5000)
        self.console.workspace.ensure_selected("TestSpace")
        self.log("Selected default workspace 'TestSpace'")

    def teardown(self) -> None:
        self.browser.close()
        self.playwright.stop()

    def determine_browser(self) -> BrowserType:
        """
        Provide random coverage for all browsers.
        """

        # SY-2928
        # Firefox failing in CI only
        # Webkit failing on Win in CI
        browsers = ["chromium"]
        selected = random.choice(browsers)
        self.log(f"Randomly selected browser: {selected}")
        browser_attr = getattr(self.playwright, selected)
        return cast(BrowserType, browser_attr)
