#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
import time
from abc import ABC, abstractmethod
from test.console.console import Console
from typing import Any, Dict, Optional, Tuple

from playwright.sync_api import Locator, Page


class SchematicSymbol(ABC):
    """Base class for all schematic symbols"""

    page: Page
    symbol: Locator
    symbol_id: str
    channel_name: str
    label: str

    def __init__(self, page: Page, symbol_id: str, channel_name: str):

        if channel_name.strip() == "":
            raise ValueError("Channel name cannot be empty")

        self.channel_name = channel_name
        self.page = page
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
        label_input = (
            self.page.locator("text=Label").locator("..").locator("input").first
        )
        label_input.fill(label)
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
            channel_button = (
                self.page.locator(f"text={input_field}")
                .locator("..")
                .locator("button")
                .first
            )
            # Click on the selector and fille channel_name
            channel_button.click()
            search_input = self.page.locator("input[placeholder*='Search']")
            search_input.click()
            search_input.fill(channel_name)
            self.page.wait_for_timeout(500)

            # Iterate through dropdown items
            channel_found = False
            item_selector = self.page.locator(".pluto-list__item").all()
            for item in item_selector:
                if item.is_visible() and channel_name in item.inner_text().strip():
                    item.click()
                    channel_found = True
                    break

            if not channel_found:
                raise RuntimeError(
                    f"Could not find channel '{channel_name}' in dropdown"
                )

    def get_properties(self) -> Dict[str, Any]:
        return {}

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

    def set_value(self, value: Any = None) -> None:

        if value is None:
            raise ValueError(f"{self.label}: Set Value cannot be None")

        self._disable_edit_mode()
        self._click_symbol()

        # Fill the input and set the value
        value_input = self.symbol.locator("input[type='number'], input").first
        value_input.fill(str(value))
        set_button = self.symbol.locator("button").filter(has_text="Set")
        set_button.click()


class Value(SchematicSymbol):
    """Schematic value/telemetry symbol"""

    channel_name: str
    notation: str
    precision: int
    averaging_window: int
    stale_color: str
    stale_timeout: int

    def __init__(self, page: Page, symbol_id: str, channel_name: str):
        super().__init__(page, symbol_id, channel_name)

    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        *,
        notation: Optional[str] = None,
        precision: Optional[int] = None,
        averaging_window: Optional[int] = None,
        stale_color: Optional[str] = None,
        stale_timeout: Optional[int] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
        """Edit Value symbol properties including channel and telemetry settings."""

        self._click_symbol()

        # Always enforce label = channel_name for easy identification
        if channel_name is not None:
            self.set_label(channel_name)

        # Navigate to Properties > Telemetry tab
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Telemetry").click()

        if channel_name is not None:
            self.set_channel("Input Channel", channel_name)

        if notation is not None:
            notation_button = (
                self.page.locator("text=Notation").locator("..").locator("button").first
            )
            notation_button.click()
            self.page.get_by_text(notation).click()

        if precision is not None:
            precision_input = (
                self.page.locator("text=Precision").locator("..").locator("input")
            )
            precision_input.fill(str(precision))
            precision_input.press("Enter")

        if averaging_window is not None:
            averaging_window_input = (
                self.page.locator("text=Averaging Window")
                .locator("..")
                .locator("input")
            )
            averaging_window_input.fill(str(averaging_window))
            averaging_window_input.press("Enter")

        if stale_color is not None:
            if not re.match(r"^#[0-9A-Fa-f]{6}$", stale_color):
                raise ValueError(
                    "stale_color must be a valid hex color (e.g., #FF5733)"
                )

            color_button = (
                self.page.locator("text=Color").locator("..").locator("button")
            )
            color_button.click()
            hex_input = self.page.locator("text=Hex").locator("..").locator("input")
            hex_input.fill(stale_color.replace("#", ""))
            hex_input.press("Enter")
            self.page.keyboard.press("Escape")

        if stale_timeout is not None:
            stale_timeout_input = (
                self.page.locator("text=Stale Timeout").locator("..").locator("input")
            )
            stale_timeout_input.fill(str(stale_timeout))
            stale_timeout_input.press("Enter")

        applied_properties: Dict[str, Any] = {}
        if channel_name is not None:
            applied_properties["channel"] = channel_name
        if notation is not None:
            applied_properties["notation"] = notation
        if precision is not None:
            applied_properties["precision"] = precision
        if averaging_window is not None:
            applied_properties["averaging_window"] = averaging_window
        if stale_color is not None:
            applied_properties["stale_color"] = stale_color
        if stale_timeout is not None:
            applied_properties["stale_timeout"] = stale_timeout

        return applied_properties

    def get_properties(self) -> Dict[str, Any]:
        """Get the current properties of the symbol"""
        self._click_symbol()
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Telemetry").click()

        props: Dict[str, Any] = {
            "channel": "",
            "notation": "",
            "precision": -1,
            "averaging_window": -1,
            "stale_color": "",
            "stale_timeout": -1,
        }

        # Channel Name
        channel_display = (
            self.page.locator("text=Input Channel").locator("..").locator("button")
        )
        if channel_display.count() > 0:
            props["channel"] = channel_display.inner_text().strip()

        # Precision
        precision_input = (
            self.page.locator("text=Precision").locator("..").locator("input")
        )
        props["precision"] = int(precision_input.input_value())

        # Averaging Window
        avg_input = (
            self.page.locator("text=Averaging Window").locator("..").locator("input")
        )
        props["averaging_window"] = int(avg_input.input_value())

        # Staleness Timeout
        timeout_input = (
            self.page.locator("text=Stale Timeout").locator("..").locator("input")
        )
        props["stale_timeout"] = int(timeout_input.input_value())

        # Notation
        notation_options = ["Scientific", "Engineering", "Standard"]
        for option in notation_options:
            try:
                button = self.page.get_by_text(option).first
                if button.count() > 0:
                    class_name = button.get_attribute("class") or ""
                    if "pluto-btn--filled" in class_name:
                        props["notation"] = str(option).lower()
                        break
            except:
                continue

        # Staleness Color - get hex value from color picker
        color_button = self.page.locator("text=Color").locator("..").locator("button")
        color_button.click()
        hex_input = self.page.locator("text=Hex").locator("..").locator("input")
        if hex_input.count() > 0:
            hex_value = hex_input.input_value()
            if hex_value:
                props["stale_color"] = (
                    f"#{hex_value}" if not hex_value.startswith("#") else hex_value
                )
        # Close color picker
        self.page.keyboard.press("Escape")

        return props


class Setpoint(SchematicSymbol):
    """Schematic setpoint/control symbol"""

    def __init__(self, page: Page, symbol_id: str, channel_name: str):
        super().__init__(page, symbol_id, channel_name)

    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
        """Edit Setpoint properties including channel settings."""

        self._click_symbol()

        applied_properties: Dict[str, Any] = {}
        if channel_name is not None:
            self.set_label(channel_name)

            # Navigate to Properties > Control tab
            self.page.get_by_text("Properties").click()
            self.page.get_by_text("Control").last.click()
            self.set_channel("Command Channel", channel_name)
            applied_properties["channel"] = channel_name

        return applied_properties


class Schematic(Console):
    """
    Parent class for schematic tests
    """

    def setup(self) -> None:
        super().setup()
        self.create_page("Schematic")
        self.page.locator(".react-flow__pane").dblclick()

    def _add_symbol_to_schematic(self, symbol_type: str) -> str:
        """
        Common logic for adding a symbol to the schematic and returning its ID

        SY-2965: Will become schematic.add_symbol()
        """

        # Go to "Symbols" tab
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

        self._log_message(f"Added setpoint with channel {channel_name}")
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

        self._log_message(f"Added value with channel {channel_name}")
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

        self._log_message(
            f"Connecting {source_symbol.label}:{source_handle} to {target_symbol.label}:{target_handle}"
        )

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
