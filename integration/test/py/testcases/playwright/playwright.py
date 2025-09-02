#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from framework.test_case import TestCase
from playwright.sync_api import sync_playwright
import time

class Playwright(TestCase):
    """
    Playwright TestCase implementation
    """

    def setup(self) -> None:
        
        headless = self.params.get("headless", True)
        default_timeout = self.params.get("default_timeout", 1000) # 1s
        default_nav_timeout = self.params.get("default_nav_timeout", 1000) # 1s

        # Open page
        self._log_message(f"Opening browser in {'headless' if headless else 'visible'} mode")
        self.playwright = sync_playwright().start()
        self.browser = self.playwright.chromium.launch(headless=headless)
        self.page = self.browser.new_page()
        
        # Set timeouts
        self.page.set_default_timeout(default_timeout)  # 1s
        self.page.set_default_navigation_timeout(default_nav_timeout)  #1s

        # Login
        self.page.goto("http://localhost:9090/", timeout=10000)
        self.page.locator('input').first.fill('synnax')
        self.page.locator('input[type="password"]').fill('seldon')
        self.page.get_by_role('button', name='Sign In').click()
        self.page.wait_for_load_state('networkidle')

        # Toggle theme
        time.sleep(0.1)
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        time.sleep(0.1)
        self.page.wait_for_selector("#toggle-theme", timeout=5000)
        self.page.locator("#toggle-theme").click()

    def teardown(self) -> None:
        self.browser.close()


    def close_page(self, page_name: str) -> None:
        tab = self.page.locator("div").filter(has_text=re.compile(f"^{re.escape(page_name)}$"))
        tab.get_by_label("pluto-tabs__close").click()

    

    def create_page(self, page_type: str, page_name: str = None) -> None:
        # Handle "a" vs "an" article for proper command matching
        vowels = ['A', 'E', 'I', 'O', 'U']
        # Special case for "NI" (en-eye)
        article = "an" if page_type[0].upper() in vowels or page_type.startswith("NI") else "a"
        page_command = f"Create {article} {page_type}"

        self.page.keyboard.press("ControlOrMeta+Shift+p")
        self.page.wait_for_selector(f"text={page_command}")
        self.page.get_by_text(page_command).click()
        
        if page_name is not None:
            tab = self.page.locator("div").filter(has_text=re.compile(f"^{re.escape(page_type)}$"))
            tab.dblclick()
            self.page.get_by_text(page_type).first.fill(page_name)
            self.page.keyboard.press("Enter")  # Confirm the change
            return page_name
        else:
            return page_type