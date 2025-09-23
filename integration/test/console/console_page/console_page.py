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

    def create(self, page_type: str, page_name: Optional[str] = None) -> None:
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

        # Execute command
        self.console.command_palette(page_command)

        # If page name provided, rename the page
        if page_name is not None:
            tab = self.page.locator("div").filter(
                has_text=re.compile(f"^{re.escape(page_type)}$")
            )
            tab.dblclick()
            self.page.get_by_text(page_type).first.fill(page_name)
            self.page.keyboard.press("Enter")  # Confirm the change

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