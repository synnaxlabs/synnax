#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import re
import time
from typing import Any, Dict, List, Optional, Union, cast, TYPE_CHECKING

from playwright.sync_api import Page, Locator

if TYPE_CHECKING:
    from ..console import Console


class ConsolePage:
    """Console page management interface"""

    def __init__(self, page: Page, console: "Console") -> None:
        self.page = page
        self.console = console

        # Page identification - subclasses should set these
        self.page_type: str = ""
        self.pluto_label: str = ""
        self.tab_locator: Optional[Locator] = None
        self.pane_locator: Optional[Locator] = None
        self.id: Optional[str] = None

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

    def _dblclick_canvas(self) -> None:
        """Double click on canvas."""
        if self.pane_locator and self.pane_locator.count() > 0:
            self.pane_locator.dblclick()
            time.sleep(0.1)

    def _click_canvas(self) -> None:
        """Single click on canvas."""
        if self.pane_locator and self.pane_locator.count() > 0:
            self.pane_locator.click()
            time.sleep(0.1)

    def new(self) -> str:
        self.tab_locator, self.id = self.console.create_page(self.page_type)
        if self.pluto_label:
            # Handler assumes only one page with label will be open.
            self.pane_locator = self.page.locator(self.pluto_label)
        self._dblclick_canvas()
        return self.id or ""

    def save_screenshot_NEEDS_WORK(self, path: Optional[str] = None) -> None:
        """Save a screenshot of the pane area with margin."""
        if path is None:
            os.makedirs("test/results", exist_ok=True)
            path = f"test/results/{self.id}.png"

        box = self.pane_locator.bounding_box()

        if box:
            self._save_with_clip_NEEDS_WORK(path, box)
        else:
            self._save_element_screenshot(path, self.pane_locator)

    def _save_with_clip_NEEDS_WORK(self, path: str, box: Dict[str, float]) -> None:
        """Save screenshot with custom clipping area."""
        margin = 10
        clip_area = {
            "x": max(0, box["x"] - margin),
            "y": max(0, box["y"] - margin),
            "width": box["width"] + 2 * margin,
            "height": box["height"] + 2 * margin,
        }

        self.page.screenshot(
            path=path,
            clip=cast(FloatRect, clip_area),
            animations="disabled",
            omit_background=False,
            type="png",
            scale="device",
        )
    def _save_element_screenshot_NEEDS_WORK(self, path: str, locator: Any) -> None:
        """Save element screenshot as fallback."""
        locator.screenshot(
            path=path,
            animations="disabled",
            omit_background=False,
            type="png",
            scale="device",
        )