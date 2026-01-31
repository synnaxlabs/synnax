#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import os
import re
import time
from typing import Any, Literal, Self, cast

import synnax as sy
from playwright.sync_api import FloatRect, Locator, Page, ViewportSize

from framework.utils import get_results_path

from .layout import LayoutClient

PageType = Literal[
    "Control Sequence",
    "Line Plot",
    "Schematic",
    "Log",
    "Table",
    "NI Analog Read Task",
    "NI Analog Write Task",
    "NI Counter Read Task",
    "NI Digital Read Task",
    "NI Digital Write Task",
    "LabJack Read Task",
    "LabJack Write Task",
    "OPC UA Read Task",
    "OPC UA Write Task",
]


class ConsolePage:
    """Console page management interface"""

    client: sy.Synnax
    page: Page
    layout: LayoutClient
    page_name: str
    page_type: str
    pluto_label: str
    tab_locator: Locator | None = None
    pane_locator: Locator | None = None
    id: str | None = None

    @classmethod
    def from_open_page(
        cls,
        layout: LayoutClient,
        client: sy.Synnax,
        name: str,
    ) -> Self:
        """Create instance from an already-opened page.

        Use this factory method when a page has already been opened (e.g., via
        drag_page_to_mosaic or open_page) and you need to create a typed wrapper.

        Args:
            layout: LayoutClient instance
            client: Synnax client instance
            name: Name of the page

        Returns:
            Instance of the page class
        """
        pane = layout.page.locator(cls.pluto_label)
        pane.first.wait_for(state="visible", timeout=5000)

        return cls(layout, client, name, pane_locator=pane.first)

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        *,
        pane_locator: Locator,
    ) -> None:
        """Initialize a page wrapper around an existing UI page.

        IMPORTANT: This constructor wraps an existing page that has already been created
        in the Console UI via Playwright automation. It does NOT create a new page.

        The separation exists because:
        - UI page creation: Handled by workspace.create_page() which clicks buttons
          and interacts with the Console UI via Playwright
        - Python wrapper creation: This constructor creates a Python object that
          provides programmatic access to the already-existing UI page

        Think of this as "wrapping" or "binding" to an existing UI element, similar to
        how Playwright locators bind to DOM elements.

        Args:
            layout: LayoutClient for UI operations (includes notifications)
            client: Synnax client instance
            page_name: Name of the existing page to wrap
            pane_locator: Playwright locator for the page's pane element.
                This locator identifies which UI page this Python object represents.
                Must be provided when creating the wrapper.
        """
        self.client = client
        self.page = layout.page
        self.layout = layout
        self.page_name = page_name
        self.pane_locator = pane_locator

    def _get_tab(self) -> Locator:
        """Get the tab locator for this page."""
        return self.layout.get_tab(self.page_name)

    def close(self) -> None:
        """
        Close a page by name.
        Ignore unsaved changes.
        """
        tab = self._get_tab()
        tab.wait_for(state="visible", timeout=5000)
        close_button = tab.get_by_label("pluto-tabs__close")
        close_button.wait_for(state="visible", timeout=5000)
        close_button.click()

        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()
        tab.wait_for(state="hidden", timeout=5000)

    @property
    def is_open(self) -> bool:
        """Check if the page tab is visible."""
        tab = self.layout.get_tab(self.page_name)
        return tab.count() > 0 and tab.is_visible()

    @property
    def is_pane_visible(self) -> bool:
        """Check if the page pane content is visible."""
        return self.pane_locator is not None and self.pane_locator.is_visible()

    def rename(self, new_name: str) -> None:
        """Rename the page by double-clicking the tab name.

        Args:
            new_name: The new name for the page
        """
        self.layout.rename_tab(old_name=self.page_name, new_name=new_name)
        self.page_name = new_name

    def _dblclick_canvas(self) -> None:
        """Double click on canvas."""
        if self.pane_locator and self.pane_locator.count() > 0:
            self.pane_locator.dblclick()

    def _click_canvas(self) -> None:
        """Single click on canvas."""
        if self.pane_locator and self.pane_locator.count() > 0:
            self.pane_locator.click()

    def _initialize_from_workspace(self, tab_locator: Locator, page_id: str) -> None:
        """Initialize page after workspace creates it.

        This is called by workspace after create_page() to set up the page instance.
        """
        self.tab_locator = tab_locator
        self.id = page_id
        if self.pluto_label:
            self.pane_locator = self.page.locator(self.pluto_label)
        self._dblclick_canvas()

    def screenshot(self, path: str | None = None) -> None:
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

    def get_title(self) -> str:
        """Get the current page title from the Properties tab.

        Returns:
            The current page title
        """
        self.page.locator("#properties").click(timeout=5000)
        return self.layout.get_input_field("Title")

    def copy_link(self) -> str:
        """Copy link to the page via the toolbar link button.

        Returns:
            The copied link from clipboard (empty string if clipboard access fails).
        """
        self.layout.notifications.close_all()
        self.layout.show_visualization_toolbar()
        link_button = self.page.locator(".pluto-icon--link").locator("..")
        link_button.click(timeout=5000)
        return self.layout.read_clipboard()

    def export_json(self) -> dict[str, Any]:
        """Export the page as a JSON file via the toolbar export button.

        The file is saved to the tests/results directory with the page name.

        Returns:
            The exported JSON content as a dictionary.
        """
        self.layout.show_visualization_toolbar()
        export_button = self.page.locator(".pluto-icon--export").locator("..")
        self.page.evaluate("delete window.showSaveFilePicker")

        with self.page.expect_download(timeout=5000) as download_info:
            export_button.click()

        download = download_info.value
        save_path = get_results_path(f"{self.page_name}.json")
        download.save_as(save_path)
        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def get_value(self, channel_name: str) -> float | None:
        """Get the latest data value for any channel using the synnax client"""
        try:
            # Retry with short delays for CI resource constraints
            for attempt in range(3):
                latest_value = self.client.read_latest(channel_name)
                if latest_value is not None and len(latest_value) > 0:
                    return float(latest_value)

                # If read_latest is empty, read recent time range
                now = sy.TimeStamp.now()
                recent_range = sy.TimeRange(now - sy.TimeSpan.SECOND * 3, now)
                frame = self.client.read(recent_range, channel_name)
                if len(frame) > 0:
                    return float(frame[-1])
                if attempt < 2:
                    sy.sleep(0.2)

            return None

        except Exception as e:
            raise RuntimeError(f'Could not get value for channel "{channel_name}": {e}')
