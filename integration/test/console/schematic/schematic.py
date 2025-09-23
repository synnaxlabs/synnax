#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Optional, Tuple, TYPE_CHECKING
from playwright.sync_api import Page, Locator

from ..console_page import ConsolePage
from .schematic_symbol import SchematicSymbol
from .setpoint import Setpoint
from .value import Value
import time

if TYPE_CHECKING:
    from ..console import Console


class Schematic(ConsolePage):
    """
    Parent class for schematic tests
    """
    locator: Locator
    id: str

    def __init__(self, page: Page, console: "Console"):
        super().__init__(page, console)
        self.locator = None
        self.id = None


    def _add_symbol_to_schematic(self, symbol_type: str) -> str:
        """
        Common logic for adding a symbol to the schematic and returning its ID

        SY-2965: Will become schematic.add_symbol()
        """

        # Go to "Symbols" tab
        self._dblclick_canvas()
        self.page.wait_for_selector("text=Symbols", timeout=10000)
        self.page.get_by_text("Symbols", exact=True).click()
        # Count existing symbols before adding
        symbols_count = len(self.page.locator("[data-testid^='rf__node-']").all())

        # Select symbol
        self.page.wait_for_selector(f"text={symbol_type}", timeout=3000)
        self.page.get_by_text(symbol_type, exact=True).first.click()

        # Wait for new symbol to appear
        self.page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {symbols_count}"
        )

        # Get all symbols and find the new one
        all_symbols = self.page.locator("[data-testid^='rf__node-']").all()
        symbol_id = (
            all_symbols[-1].get_attribute("data-testid") or "unknown"
        )  # Last one should be the newest symbol

        return symbol_id

    def create_setpoint(self, channel_name: str) -> Setpoint:
        """Create a setpoint symbol on the schematic"""

        setpoint_id = self._add_symbol_to_schematic("Setpoint")
        setpoint = Setpoint(self.page, setpoint_id, channel_name)
        setpoint.edit_properties(channel_name=channel_name)

        #self._log_message(f"Added setpoint with channel {channel_name}")
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
        """Create a value symbol on the schematic"""
        value_id = self._add_symbol_to_schematic("Value")
        value = Value(self.page, value_id, channel_name)

        value.edit_properties(
            channel_name=channel_name,
            notation=notation,
            precision=precision,
            averaging_window=averaging_window,
            stale_color=stale_color,
            stale_timeout=stale_timeout,
        )

        #self._log_message(f"Added value with channel {channel_name}")
        return value

    def connect_symbols(
        self,
        source_symbol: SchematicSymbol,
        source_handle: str,
        target_symbol: SchematicSymbol,
        target_handle: str,
    ) -> None:
        """
        Connect two symbols by dragging from source handle to target handle.
        """
        source_x, source_y = self.find_symbol_handle(source_symbol, source_handle)
        target_x, target_y = self.find_symbol_handle(target_symbol, target_handle)

        #self._log_message(
        #    f"Connecting {source_symbol.label}:{source_handle} to {target_symbol.label}:{target_handle}"
        #)

        self.page.mouse.move(source_x, source_y)
        self.page.mouse.down()
        self.page.mouse.move(target_x, target_y, steps=10)
        self.page.mouse.up()

    def find_symbol_handle(
        self, symbol: SchematicSymbol, handle: str
    ) -> Tuple[float, float]:
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
            raise ValueError(
                f"Invalid handle: {handle}. Must be one of {list(handle_positions.keys())}"
            )

        return handle_positions[handle]

    def _dblclick_canvas(self) -> None:
        """ 
        Double clicks on canvas. 
        Might move to ConsolePage.
        """
        canvas = self.page.locator(".react-flow__pane") # Not specifically this
        if canvas.count() > 0:
            canvas.dblclick()
            time.sleep(0.1)

    def _click_canvas(self) -> None:
        """ 
        Double clicks on canvas. 
        Might move to ConsolePage.
        """
        canvas = self.page.locator(".react-flow__pane") # Not specifically this
        if canvas.count() > 0:
            canvas.click()
            time.sleep(0.1)

    def new(self) -> str:
        self.locator, self.id = self.console.create_page("Schematic")
        self._dblclick_canvas()