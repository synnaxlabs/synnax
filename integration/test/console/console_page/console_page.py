#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from typing import Optional, TYPE_CHECKING

from playwright.sync_api import Page

if TYPE_CHECKING:
    from ..console import Console


class ConsolePage:
    """Console pages management interface"""

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console

    def create(self, page_type: str, page_name: Optional[str] = None) -> str:
        """Create a new page via command palette and return the page ID"""
        # Handle "a" vs "an" article for proper command matching
        vowels = ["A", "E", "I", "O", "U"]
        # Special case for "NI" (en-eye)
        article = (
            "an"
            if page_type[0].upper() in vowels or page_type.startswith("NI")
            else "a"
        )
        page_command = f"Create {article} {page_type}"

        # Execute command
        self.console.command_palette(page_command)

        # Wait for page to be created - use a simple timeout approach
        self.page.wait_for_timeout(1000)  # Give time for page creation

        # Try to find the newly created page/tab by page_type text
        try:
            # Look for the page type text which should appear after creation
            page_tab = self.page.locator("div").filter(
                has_text=re.compile(f"^{re.escape(page_type)}$")
            ).first
            page_id = page_tab.inner_text().strip()
        except Exception:
            # Fallback: generate a simple ID
            import time
            page_id = f"{page_type}_{int(time.time() * 1000) % 10000}"

        # If page name provided, rename the page
        if page_name is not None:
            tab = self.page.locator("div").filter(
                has_text=re.compile(f"^{re.escape(page_type)}$")
            )
            tab.dblclick()
            self.page.get_by_text(page_type).first.fill(page_name)
            self.page.keyboard.press("Enter")  # Confirm the change
            page_id = page_name  # Update page_id to the custom name

        return page_id or f"{page_type}_{existing_tabs + 1}"

    def close(self, page_name: str) -> None:
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