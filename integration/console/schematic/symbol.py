#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import platform
from abc import ABC, abstractmethod
from collections.abc import Generator
from contextlib import contextmanager
from dataclasses import dataclass
from typing import Any

import synnax as sy
from playwright.sync_api import Locator, Page

from ..console import Console


@dataclass
class Box:
    """Bounding box with computed properties for position and dimensions."""

    x: float  # left edge
    y: float  # top edge
    width: float
    height: float

    @property
    def left(self) -> float:
        """Left edge x-coordinate."""
        return self.x

    @property
    def right(self) -> float:
        """Right edge x-coordinate."""
        return self.x + self.width

    @property
    def top(self) -> float:
        """Top edge y-coordinate."""
        return self.y

    @property
    def bottom(self) -> float:
        """Bottom edge y-coordinate."""
        return self.y + self.height

    @property
    def center_x(self) -> float:
        """Center x-coordinate."""
        return self.x + self.width / 2

    @property
    def center_y(self) -> float:
        """Center y-coordinate."""
        return self.y + self.height / 2


class Symbol(ABC):
    """Base class for all schematic symbols"""

    page: Page
    console: Console
    symbol: Locator
    symbol_id: str
    channel_name: str
    label: str
    rotatable: bool

    def __init__(
        self,
        page: Page,
        console: Console,
        symbol_id: str,
        channel_name: str,
        rotatable: bool = False,
    ):
        if channel_name.strip() == "":
            raise ValueError("Channel name cannot be empty")

        self.channel_name = channel_name
        self.page = page
        self.console = console
        self.symbol_id = symbol_id
        self.label = channel_name
        self.rotatable = rotatable

        self.symbol = self.page.get_by_test_id(self.symbol_id)
        self.set_label(channel_name)

    @contextmanager
    def bring_to_front(self, element: Locator) -> Generator[Locator, None, None]:
        original_z_index = element.evaluate("element => element.style.zIndex || 'auto'")
        element.evaluate("element => element.style.zIndex = '9999'")
        try:
            yield element
        finally:
            element.evaluate(f"element => element.style.zIndex = '{original_z_index}'")

    def _disable_edit_mode(self) -> None:
        edit_off_icon = self.page.get_by_label("pluto-icon--edit-off")
        if edit_off_icon.count() > 0:
            edit_off_icon.click()

    def _click_symbol(self) -> None:
        with self.bring_to_front(self.symbol) as s:
            s.click(force=True)
        self.console.page.wait_for_timeout(100)

    def meta_click(self) -> None:
        """Click the symbol with the platform-appropriate modifier key (Cmd/Ctrl) held down."""

        modifier = "Meta" if platform.system() == "Darwin" else "Control"
        self.page.keyboard.down(modifier)
        with self.bring_to_front(self.symbol) as s:
            s.click(force=True)
        self.console.page.wait_for_timeout(100)
        self.page.keyboard.up(modifier)

    def set_label(self, label: str) -> None:
        self._click_symbol()
        self.page.get_by_text("Style").click(force=True)
        self.console.fill_input_field("Label", label)
        self.label = label

    @abstractmethod
    def edit_properties(
        self,
        channel_name: str | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """
        Edit symbol properties. Must be implemented by all child classes.

        Args:
            channel_name: Optional channel name to set
            **kwargs: Additional properties specific to each symbol type

        Returns:
            Dictionary of applied properties
        """
        pass

    def set_channel(self, input_field: str, channel_name: str) -> None:
        if channel_name is not None:
            self.console.click_btn(input_field)
            self.console.select_from_dropdown(channel_name, "Search")

    def set_value(self, value: float) -> None:
        """Set the symbol's value if applicable. Default implementation does nothing."""
        pass

    def move(self, delta_x: int, delta_y: int) -> None:
        """Move the symbol by the specified number of pixels using drag"""
        pos = self.position
        start_x = pos.center_x
        start_y = pos.center_y
        target_x = start_x + delta_x
        target_y = start_y + delta_y

        # Move
        self.page.mouse.move(start_x, start_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()
        sy.sleep(0.1)  # CI flakiness

    @property
    def position(self) -> Box:
        """
        Get the symbol's bounding box information for alignment checks.

        Returns:
            Box dataclass with:
                - x: x coordinate of the left edge
                - y: y coordinate of the top edge
                - width: width of the bounding box
                - height: height of the bounding box

            Computed properties available:
                - left: x coordinate of the left edge (same as x)
                - right: x coordinate of the right edge
                - top: y coordinate of the top edge (same as y)
                - bottom: y coordinate of the bottom edge
                - center_x: x coordinate of the center (for horizontal alignment)
                - center_y: y coordinate of the center (for vertical alignment)

        Note: Edge alignments (left/right/top/bottom) use bounding box edges.
              Center alignments (center_x/center_y) use the box center as a proxy for handle positions.

        Raises:
            RuntimeError: If the symbol's bounding box cannot be retrieved
        """
        box = self.symbol.bounding_box()
        if not box:
            raise RuntimeError(
                f"Could not get bounding box for symbol {self.symbol_id}"
            )

        return Box(
            x=box["x"],
            y=box["y"],
            width=box["width"],
            height=box["height"],
        )

    def delete(self) -> None:
        self._click_symbol()
        self.console.DELETE

    def toggle_absolute_control(self) -> None:
        """Toggle absolute control authority for this symbol by clicking its control chip button."""
        # Locate the control chip button within this specific symbol's container
        control_chip = self.symbol.locator(".pluto-control-chip").first

        # Bring to front and click to ensure visibility and interaction
        with self.bring_to_front(control_chip) as chip:
            chip.click(force=True)

        self.console.page.wait_for_timeout(100)

    def get_properties(self, tab: str = "Symbols") -> dict[str, Any]:
        """
        Get the current properties of the symbol.

        Base implementation opens the Properties panel and optionally selects a tab.
        Subclasses should call super().get_properties(tab="TabName") to open the correct tab.

        Args:
            tab: Optional tab name to select (e.g., "Control", "Telemetry", "Style")

        Returns:
            Empty dict - subclasses should populate with actual properties
        """
        self._click_symbol()
        self.page.get_by_text("Properties").click()
        if tab:
            self.page.get_by_text(tab).last.click()
        return {}
