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

from playwright.sync_api import (
    Browser,
    BrowserContext,
    BrowserType,
    Page,
    sync_playwright,
)
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from console.console import Console
from framework.models import STATUS
from framework.run_dir import resolve_results_path
from framework.test_case import TestCase


class ConsoleCase(TestCase):
    """
    Console TestCase implementation using Playwright

    Environment Variables:
    - PLAYWRIGHT_CONSOLE_HEADED: Run in headed mode (default: False)
      Can be set via command line: --console-headed or -ch
    """

    browser: Browser
    context: BrowserContext
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
        self.context = self.browser.new_context(
            permissions=["clipboard-read", "clipboard-write"],
        )

        # Tracing is started immediately so that a partial trace is captured even
        # if setup itself fails. The trace is discarded on pass and persisted to
        # the results directory on FAILED / TIMEOUT / KILLED outcomes.
        self.context.tracing.start(
            name=self.name,
            screenshots=True,
            snapshots=True,
            sources=True,
        )

        self.page = self.context.new_page()

        # Set timeouts
        self.page.set_default_timeout(default_timeout)  # 1s
        self.page.set_default_navigation_timeout(default_nav_timeout)  # 1s

        # Try embedded console first, fallback to dev server if no console found
        host = self.synnax_connection.server_address
        port = self.synnax_connection.port

        self.page.goto(f"http://{host}:{port}/", timeout=20000)
        if "core built without embedded console" in self.page.content().lower():
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

        self.page.get_by_role("button", name="Log In").click(timeout=2000)

        self.console = Console(self.page, self.client)

        # Initialized signal (Not "Get Started" page)
        self.page.wait_for_selector(
            ".console-palette button", state="visible", timeout=10000
        )

        # Default workspace for all tests
        self.console.workspace.ensure_selected("TestSpace")

        # Prevent state pollution
        # Selecting workspace restores tabs
        self.console.close_all_tabs()
        self.console.notifications.close_connection()
        self._cleanup_pages: list[str] = []

    def teardown(self) -> None:
        # Stop and persist the trace before cleanup; cleanup may mutate page state
        # in ways that mask the failure we want to capture.
        self._stop_tracing()

        # setup() may fail before _cleanup_pages is assigned; tolerate that here
        # so teardown still closes the browser instead of raising.
        if getattr(self, "_cleanup_pages", None):
            try:
                self.console.workspace.delete_pages(self._cleanup_pages)
            except PlaywrightTimeoutError:
                pass
        self.context.close()
        self.browser.close()
        self.playwright.stop()

    def _stop_tracing(self) -> None:
        failed_states = (STATUS.FAILED, STATUS.TIMEOUT, STATUS.KILLED)
        try:
            if self._status in failed_states:
                trace_path = resolve_results_path("trace.zip")
                self.context.tracing.stop(path=trace_path)
                self.log(f"Trace saved: {trace_path}")
            else:
                self.context.tracing.stop()
        except Exception as e:
            # Tracing teardown must never break test teardown. The trace is a
            # debugging aid, not a correctness requirement.
            self.log(f"Failed to stop trace: {e}")

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
