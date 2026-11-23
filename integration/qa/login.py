#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Logs the Chromium at CDP_PORT in to the Console and opens the sample project.

Usage: uv run python qa/login.py CORE_PORT CDP_PORT
"""

import sys

from playwright.sync_api import sync_playwright


def main(core_port: int, cdp_port: int) -> None:
    with sync_playwright() as p:
        browser = p.chromium.connect_over_cdp(f"http://localhost:{cdp_port}")
        context = browser.contexts[0]
        page = context.pages[0] if context.pages else context.new_page()
        page.goto(f"http://localhost:{core_port}/", timeout=60_000)
        page.get_by_label("Username", exact=True).first.fill("synnax")
        page.get_by_label("Password", exact=True).first.fill("seldon")
        page.get_by_role("button", name="Log in").click()
        page.get_by_text("Sample project", exact=True).click(timeout=30_000)


if __name__ == "__main__":
    main(int(sys.argv[1]), int(sys.argv[2]))
