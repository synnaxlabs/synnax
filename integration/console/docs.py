#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

from playwright.sync_api import Page

from console.base import ResourceClient

URL = "https://docs.synnaxlabs.com/reference/console/get-started"

_URL_PATTERN = re.compile(r"^https://docs\.synnaxlabs\.com/")


class DocsClient(ResourceClient):
    """Client for the Console's documentation actions.

    Both actions hand the docs URL to the browser, so each returns the tab it
    opened. The caller owns closing it.
    """

    def open_via_command_palette(self) -> Page:
        """Open the documentation via the command palette."""
        with self.page.context.expect_page() as info:
            self.layout.command_palette("Read documentation")
        return self._settle(info.value)

    def open_via_question_mark_icon(self) -> Page:
        """Open the documentation by clicking the question mark icon."""
        btn = self.page.locator(".console-docs__open-button")
        btn.wait_for(state="visible", timeout=5000)
        with self.page.context.expect_page() as info:
            btn.click()
        return self._settle(info.value)

    def _settle(self, tab: Page) -> Page:
        """Wait for ``tab`` to point at the docs site.

        The assertion is on the requested URL, not on a completed load: the
        docs site is external and must not gate the test on its reachability.
        """
        tab.wait_for_url(_URL_PATTERN, wait_until="commit", timeout=10000)
        return tab
