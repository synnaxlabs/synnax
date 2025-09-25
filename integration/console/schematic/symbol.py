#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from abc import ABC, abstractmethod
from typing import TYPE_CHECKING, Any, Dict, Optional

from playwright.sync_api import Locator, Page

if TYPE_CHECKING:
    from ..console import Console


class Symbol(ABC):
    """Base class for all schematic symbols"""

    page: Page
    console: "Console"
    symbol: Locator
    symbol_id: str
    channel_name: str
    label: str

    def __init__(
        self, page: Page, console: "Console", symbol_id: str, channel_name: str
    ):

        if channel_name.strip() == "":
            raise ValueError("Channel name cannot be empty")

        self.channel_name = channel_name
        self.page = page
        self.console = console
        self.symbol_id = symbol_id
        self.label = channel_name

        self.symbol = self.page.get_by_test_id(self.symbol_id)
        self.set_label(channel_name)

    def _disable_edit_mode(self) -> None:
        edit_off_icon = self.page.get_by_label("pluto-icon--edit-off")
        if edit_off_icon.count() > 0:
            edit_off_icon.click()

    def _click_symbol(self) -> None:
        self.symbol.click(force=True)
        time.sleep(0.1)

    def set_label(self, label: str) -> None:
        self._click_symbol()
        self.page.get_by_text("Style").click()
        self.console.fill_input_field("Label", label)
        self.label = label

    @abstractmethod
    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
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

    def move(self, delta_x: int, delta_y: int) -> None:
        """Move the symbol by the specified number of pixels using drag"""
        box = self.symbol.bounding_box()
        if not box:
            raise RuntimeError(
                f"Could not get bounding box for symbol {self.symbol_id}"
            )

        # Calculate target position
        start_x = box["x"] + box["width"] / 2
        start_y = box["y"] + box["height"] / 2
        target_x = start_x + delta_x
        target_y = start_y + delta_y

        # Move
        self.page.mouse.move(start_x, start_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()

        # Verify the move
        new_box = self.symbol.bounding_box()
        if not new_box:
            raise RuntimeError(
                f"Could not get new bounding box for symbol {self.symbol_id}"
            )

        final_x = new_box["x"] + new_box["width"] / 2
        final_y = new_box["y"] + new_box["height"] / 2

        grid_tolerance = 25
        if (
            abs(final_x - target_x) > grid_tolerance
            or abs(final_y - target_y) > grid_tolerance
        ):
            raise RuntimeError(
                f"Symbol {self.symbol_id} moved to ({final_x}, {final_y}) instead of ({target_x}, {target_y})"
            )
