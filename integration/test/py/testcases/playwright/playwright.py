#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import sys
import time

import synnax as sy

from framework.test_case import SynnaxConnection, TestCase
from playwright.sync_api import sync_playwright, Playwright


class Playwright(TestCase):
    """
    Playwright test case
    """

    def setup(self) -> None:
        """
        Setup the test case.
        """

        ''' 
        TODO: 
        - make browser a testcase arg
        - Use Playwright Github actions in CI instructions
           - Caches browser downloads
           - Handles system deps
           - Optimized for github's runner envs


        - name: Install Python dependencies
            run: poetry install

        - name: Install Playwright browsers
            uses: microsoft/playwright-github-action@v1
            # or use: run: poetry run playwright install

        '''

        self.playwright = sync_playwright().start()
        self.browser = self.playwright.chromium.launch(headless=False, slow_mo=100)
        self.console = self.browser.new_page()
        self.console.goto("http://localhost:5173/")

        super().setup()

    def run(self) -> None:
        """
        Run the test case.
        """

        # wait for page to load
        self.console.get_by_role('textbox', name='synnax').fill('synnax')
        self.console.get_by_role('textbox', name='seldon').click()
        self.console.get_by_role('textbox', name='seldon').fill('seldon')
        self.console.get_by_role('button', name='Sign In').click()
        time.sleep(10)

        # click button
        self.console.get_by_role("button", name="Sign In").click()

        # load config file
        # click button
        # load config file
        # Stop config file

    def teardown(self) -> None:
        """
        Teardown the test case.
        """

        self.browser.close()
        
        # Always call super() last
        super().teardown()
