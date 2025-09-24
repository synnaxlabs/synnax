#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Optional, Tuple

from playwright.sync_api import Page

from ..console_page import ConsolePage
from .setpoint import Setpoint
from .symbol import Symbol
from .value import Value

if TYPE_CHECKING:
    from test.console.console import Console


class Schematic(ConsolePage):
    """Schematic page management interface"""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "Schematic"
        self.pluto_label = ".react-flow__pane"

    def _add_symbol(self, symbol_type: str) -> str:
        """Add a symbol to the schematic and return its ID."""
        self._dblclick_canvas()
        self._open_symbols_tab()

        initial_count = self._count_symbols()
        self._select_symbol_type(symbol_type)
        self._wait_for_new_symbol(initial_count)

        return self._get_newest_symbol_id()

    def _open_symbols_tab(self) -> None:
        """Open the Symbols tab."""
        self.page.wait_for_selector("text=Symbols", timeout=10000)
        self.page.get_by_text("Symbols", exact=True).click()

    def _count_symbols(self) -> int:
        """Count number of symbols on the schematic."""
        return len(self.page.locator("[data-testid^='rf__node-']").all())

    def _select_symbol_type(self, symbol_type: str) -> None:
        """Select a symbol type from the symbols panel."""
        self.page.wait_for_selector(f"text={symbol_type}", timeout=3000)
        self.page.get_by_text(symbol_type, exact=True).first.click()

    def _wait_for_new_symbol(self, initial_count: int) -> None:
        """Wait for a new symbol to appear."""
        self.page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {initial_count}"
        )

    def _get_newest_symbol_id(self) -> str:
        """Get the ID of the newest symbol."""
        all_symbols = self.page.locator("[data-testid^='rf__node-']").all()
        return all_symbols[-1].get_attribute("data-testid") or "unknown"

    def create_setpoint(self, channel_name: str) -> Setpoint:
        """Create a setpoint symbol on the schematic."""
        setpoint_id = self._add_symbol("Setpoint")
        setpoint = Setpoint(self.page, self.console, setpoint_id, channel_name)
        setpoint.edit_properties(channel_name=channel_name)
        return setpoint

    def create_value(
        self,
        channel_name: str,
        notation: Optional[str] = None,
        precision: Optional[int] = None,
        averaging_window: Optional[int] = None,
        stale_color: Optional[str] = None,
        stale_timeout: Optional[int] = None,
    ) -> Value:
        """Create a value symbol on the schematic."""
        value_id = self._add_symbol("Value")
        value = Value(self.page, self.console, value_id, channel_name)

        value.edit_properties(
            channel_name=channel_name,
            notation=notation,
            precision=precision,
            averaging_window=averaging_window,
            stale_color=stale_color,
            stale_timeout=stale_timeout,
        )

        return value

    def connect_symbols(
        self,
        source_symbol: Symbol,
        source_handle: str,
        target_symbol: Symbol,
        target_handle: str,
    ) -> None:
        """Connect two symbols by dragging from source handle to target handle."""
        source_x, source_y = self.find_symbol_handle(source_symbol, source_handle)
        target_x, target_y = self.find_symbol_handle(target_symbol, target_handle)

        self.page.mouse.move(source_x, source_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()

    def find_symbol_handle(self, symbol: Symbol, handle: str) -> Tuple[float, float]:
        """Calculate the coordinates of a symbol's connection handle."""
        symbol_box = symbol.symbol.bounding_box()
        if not symbol_box:
            raise RuntimeError(f"Could not get bounding box for symbol {symbol.label}")

        x, y, w, h = (
            symbol_box["x"],
            symbol_box["y"],
            symbol_box["width"],
            symbol_box["height"],
        )

        handle_positions = {
            "left": (x, y + h / 2),
            "right": (x + w, y + h / 2),
            "top": (x + w / 2, y),
            "bottom": (x + w / 2, y + h),
        }

        if handle not in handle_positions:
            valid_handles = ", ".join(handle_positions.keys())
            raise ValueError(
                f"Invalid handle: {handle}. Must be one of: {valid_handles}"
            )

        return handle_positions[handle]
