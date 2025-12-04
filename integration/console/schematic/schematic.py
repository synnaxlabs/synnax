#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal

import synnax as sy

from console.console import Console

from ..page import ConsolePage
from .button import Button
from .setpoint import Setpoint
from .symbol import Symbol
from .value import Value
from .valve import Valve

PropertyDict = dict[str, float | str | bool]


class Schematic(ConsolePage):
    """Schematic page management interface"""

    page_type: str = "Schematic"
    pluto_label: str = ".react-flow__pane"

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
        self.console.click("Symbols")

    def _count_symbols(self) -> int:
        """Count number of symbols on the schematic."""
        return len(self.page.locator("[data-testid^='rf__node-']").all())

    def _select_symbol_type(self, symbol_type: str) -> None:
        """Select a symbol type from the symbols panel."""

        if symbol_type == "Valve":
            self.console.click("Valves")
            self.console.click("Generic")
        else:
            self.console.click("General")
            self.console.click(symbol_type)

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

    def create_button(
        self,
        channel_name: str,
        activation_delay: float | None = None,
        show_control_chip: bool | None = None,
        mode: (
            Literal["fire", "momentary", "pulse", "Fire", "Momentary", "Pulse"] | None
        ) = None,
    ) -> Button:
        """Create a button symbol on the schematic."""
        button_id = self._add_symbol("Button")
        button = Button(self.page, self.console, button_id, channel_name)
        button.edit_properties(
            channel_name=channel_name,
            activation_delay=activation_delay,
            show_control_chip=show_control_chip,
            mode=mode,
        )
        return button

    def create_valve(
        self,
        channel_name: str,
        show_control_chip: bool | None = None,
        no_state_channel: bool = False,
    ) -> Valve:
        """Create a button symbol on the schematic.
        channel_name will be used for _state and _cmd channels.
        show_control_chip is whether to show the control chip.
        """

        if not no_state_channel:
            ch_state = f"{channel_name}_state"
            ch_cmd = f"{channel_name}_cmd"
        else:
            ch_state = channel_name
            ch_cmd = channel_name

        valve_id = self._add_symbol("Valve")
        valve = Valve(self.page, self.console, valve_id, channel_name)
        valve.edit_properties(
            state_channel=ch_state,
            command_channel=ch_cmd,
            show_control_chip=show_control_chip,
        )
        return valve

    def create_value(
        self,
        channel_name: str,
        notation: str | None = None,
        precision: int | None = None,
        averaging_window: int | None = None,
        stale_color: str | None = None,
        stale_timeout: int | None = None,
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

    def find_symbol_handle(self, symbol: Symbol, handle: str) -> tuple[float, float]:
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

    def set_authority(self, authority: int) -> None:
        """Set the control authority for the schematic page."""
        if authority > 255 or authority < 0:
            raise ValueError(
                f"Control Authority must be between 0 and 255, got {authority}"
            )
        self.console.click("Control")
        self.console.fill_input_field("Control Authority", str(authority))

    def edit_properties(
        self,
        control_authority: int | None = None,
        show_control_legend: bool | None = None,
    ) -> None:
        """Edit schematic properties."""
        self.console.click("Control")

        if control_authority is not None:
            if control_authority < 0 or control_authority > 255:
                raise ValueError(
                    f"Control Authority must be between 0 and 255, got {control_authority}"
                )
            self.console.fill_input_field("Control Authority", str(control_authority))
            self.page.keyboard.press("Enter")

        if show_control_legend is not None:
            legend_toggle = (
                self.page.locator("text=Show Control State Legend")
                .locator("..")
                .locator("input[type='checkbox']")
            )
            if legend_toggle.count() > 0:
                current_state = legend_toggle.is_checked()
                if current_state != show_control_legend:
                    legend_toggle.click()

    def get_control_status(self) -> bool:
        """Get whether control is currently acquired for this schematic."""
        control_button = (
            self.page.locator(".console-controls button")
            .filter(has=self.page.locator("svg.pluto-icon--circle"))
            .first
        )

        if control_button.count() > 0:
            class_attr = control_button.get_attribute("class") or ""
            has_filled = "pluto-btn--filled" in class_attr
            return has_filled

        return False

    def acquire_control(self) -> None:
        """Acquire control of the schematic if not already acquired."""
        if not self.get_control_status():
            control_button = (
                self.page.locator(".console-controls button.pluto-btn--outlined")
                .filter(has=self.page.locator("svg.pluto-icon--circle"))
                .first
            )
            if control_button.count() > 0:
                control_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--filled", timeout=2000
                )

    def release_control(self) -> None:
        """Release control of the schematic if currently acquired."""
        if self.get_control_status():
            control_button = (
                self.page.locator(".console-controls button.pluto-btn--filled")
                .filter(has=self.page.locator("svg.pluto-icon--circle"))
                .first
            )
            if control_button.count() > 0:
                control_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--outlined", timeout=1000
                )

    def get_edit_status(self) -> bool:
        """Get whether edit is currently enabled for this schematic."""
        edit_button = (
            self.page.locator(".console-controls button")
            .filter(has=self.page.locator("svg.pluto-icon--edit"))
            .first
        )

        if edit_button.count() == 0:
            edit_button = (
                self.page.locator(".console-controls button")
                .filter(has=self.page.locator("svg.pluto-icon--edit-off"))
                .first
            )

        if edit_button.count() > 0:
            class_attr = edit_button.get_attribute("class") or ""
            has_filled = "pluto-btn--filled" in class_attr
            return has_filled

        return False

    def enable_edit(self) -> None:
        """Enable edit for the schematic if not already enabled."""
        if not self.get_edit_status():
            edit_button = (
                self.page.locator(".console-controls button.pluto-btn--outlined")
                .filter(has=self.page.locator("svg.pluto-icon--edit"))
                .first
            )
            if edit_button.count() > 0:
                edit_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--filled", timeout=2000
                )

    def disable_edit(self) -> None:
        """Disable edit for the schematic if currently enabled."""
        if self.get_edit_status():
            edit_button = (
                self.page.locator(".console-controls button.pluto-btn--filled")
                .filter(has=self.page.locator("svg.pluto-icon--edit"))
                .first
            )
            if edit_button.count() > 0:
                edit_button.click()
                self.page.wait_for_selector(
                    ".console-controls button.pluto-btn--outlined", timeout=2000
                )

    def assert_setpoint(
        self, setpoint_symbol: Setpoint, channel_name: str, value: float
    ) -> None:
        """Assert that setting the setpoint value results in the expected value in the Core."""
        setpoint_symbol.set_value(value)
        actual_value = self.get_value(channel_name)
        assert (
            actual_value == value
        ), f"Setpoint value mismatch!\nActual: {actual_value}\nExpected: {value}"

    def assert_symbol_properties(
        self, symbol: Symbol, expected_props: PropertyDict
    ) -> None:
        actual_props = symbol.get_properties()
        assert (
            actual_props == expected_props
        ), f"Props mismatch!\nActual: {actual_props}\nExpected: {expected_props}"

    def get_properties(self) -> tuple[int, bool]:
        """Get the current properties of the schematic.

        Returns:
            Tuple of (control_authority, show_control_legend)
        """
        self.console.click("Control")

        control_authority = int(self.console.get_input_field("Control Authority"))

        try:
            show_control_legend = self.console.get_toggle("Show Control State Legend")
        except Exception:
            show_control_legend = True  # Default if not found

        return (control_authority, show_control_legend)

    def assert_properties(
        self, control_authority: int = 1, show_control_legend: bool = True
    ) -> None:
        """Assert the schematic properties match expected values."""
        if control_authority < 0 or control_authority > 255:
            raise ValueError(
                f"Control Authority must be between 0 and 255, got {control_authority}"
            )

        actual_authority, actual_legend = self.get_properties()

        assert actual_authority == control_authority, (
            f"Control Authority mismatch!\n"
            f"Actual: {actual_authority}\n"
            f"Expected: {control_authority}"
        )

        assert actual_legend == show_control_legend, (
            f"Show Control Legend mismatch!\n"
            f"Actual: {actual_legend}\n"
            f"Expected: {show_control_legend}"
        )

    def assert_control_status(self, expected: bool) -> None:
        """Assert the control status matches the expected value."""
        actual = self.get_control_status()
        assert actual == expected, (
            f"Control status mismatch!\n" f"Actual: {actual}\n" f"Expected: {expected}"
        )

    def assert_control_legend_visible(self, expected: bool) -> None:
        """Assert the control state legend visibility matches the expected value."""
        legend = self.page.locator(".pluto-legend")
        is_visible = legend.count() > 0 and legend.is_visible()
        assert is_visible == expected, (
            f"Control legend visibility mismatch!\n"
            f"Actual: {is_visible}\n"
            f"Expected: {expected}"
        )

    def assert_edit_status(self, expected: bool) -> None:
        """Assert the edit status matches the expected value."""
        actual = self.get_edit_status()
        assert actual == expected, (
            f"Edit status mismatch!\n" f"Actual: {actual}\n" f"Expected: {expected}"
        )
