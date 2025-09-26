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
from typing import TYPE_CHECKING, Literal, Optional, cast

from playwright.sync_api import FloatRect, Locator, Page, ViewportSize

if TYPE_CHECKING:
    from .console import Console, PageType


class ConsolePage:
    """Console page management interface"""

    def __init__(self, page: Page, console: "Console") -> None:
        self.page = page
        self.console = console

        # Page identification - subclasses should set these
        self.page_type: "PageType" = "Log"  # Default, overridden by subclasses
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

    def screenshot(self, path: Optional[str] = None) -> None:
        """Save a screenshot of the pane area with margin."""
        if path is None:
            results_dir = os.path.join(
                os.path.dirname(__file__), "..", "tests", "results"
            )
            os.makedirs(results_dir, exist_ok=True)
            path = os.path.join(results_dir, f"{self.id}.png")

        if not self.pane_locator:
            raise RuntimeError("No pane locator available for screenshot")

        box = self.pane_locator.bounding_box()
        if not box:
            raise RuntimeError("Could not get pane bounding box for screenshot")
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

    def move(
        self,
        direction: Literal[
            "left",
            "right",
            "top",
            "bottom",
            "top-left",
            "top-right",
            "bottom-left",
            "bottom-right",
        ],
    ) -> None:
        """Move the page tab to create a split pane in the specified direction."""

        if not self.tab_locator:
            raise RuntimeError("No tab locator available for moving")

        # Get the tab's current position
        tab_box = self.tab_locator.bounding_box()
        if not tab_box:
            raise RuntimeError("Could not get tab bounding box")

        # Calculate drop position based on viewport and direction
        viewport_size = self.page.viewport_size
        if not viewport_size:
            raise RuntimeError("Could not get viewport size")
        drop_x, drop_y = self._calculate_drop_position(direction, viewport_size)

        self._drag_tab_to_position(tab_box, drop_x, drop_y)

    def _calculate_drop_position(
        self, direction: str, viewport_size: ViewportSize
    ) -> tuple[float, float]:
        """Calculate the drop position based on direction and viewport."""
        width, height = viewport_size["width"], viewport_size["height"]

        # Define drop zones near the edges for split pane creation
        margin_tb = 200  # Distance from edge to trigger split
        margin_lr = 100  # Distance from edge to trigger split
        positions = {
            "left": (margin_lr, height // 2),
            "right": (width - margin_lr, height // 2),
            "top": (width // 2, margin_tb),
            "bottom": (width // 2, height - margin_tb),
            # Corner positions
            "top-left": (margin_lr, margin_tb),
            "top-right": (width - margin_lr, margin_tb),
            "bottom-left": (margin_lr, height - margin_tb),
            "bottom-right": (width - margin_lr, height - margin_tb),
        }

        return positions[direction]

    def _drag_tab_to_position(
        self, tab_box: FloatRect, drop_x: float, drop_y: float
    ) -> None:
        """Perform the actual drag and drop operation."""
        # Start position (center of tab)
        start_x = tab_box["x"] + tab_box["width"] / 2
        start_y = tab_box["y"] + tab_box["height"] / 2

        # Perform drag and drop
        self.page.mouse.move(start_x, start_y)
        self.page.mouse.down()
        self.page.mouse.move(drop_x, drop_y, steps=10)
        time.sleep(0.2)
        self.page.mouse.up()
        # Wait for the UI
        time.sleep(0.5)
