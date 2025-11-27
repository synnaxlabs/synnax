#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal

from playwright.sync_api import Page

from console.console import Console

from .button import Button
from .setpoint import Setpoint
from .symbol import Symbol
from .value import Value
from .valve import Valve
from .valve_threeway import ValveThreeWay
from .valve_threeway_ball import ValveThreeWayBall


class SchematicSymbolFactory:
    """Factory for creating and configuring schematic symbols.

    Provides methods to create different symbol types with their specific parameters.

    Example:
        valve = schematic.valve(
            label="Pressure Valve",
            state_channel="press_vlv_state",
            command_channel="press_vlv_cmd"
        )
        valve.move(-90, -100)
    """

    def __init__(self, page: Page, console: Console):
        """Initialize the factory with page and console references.

        Args:
            page: Playwright Page instance for the schematic
            console: Console instance for UI interactions
        """
        self._page = page
        self._console = console

    def _add_symbol(self, symbol_type: str) -> str:
        """Add a symbol to the schematic and return its ID.

        Args:
            symbol_type: The type of symbol to add (e.g., "Valve", "Button", "Three Way")

        Returns:
            The symbol ID of the newly created symbol
        """
        self._open_symbols_tab()

        initial_count = self._count_symbols()
        self._select_symbol_type(symbol_type)
        self._wait_for_new_symbol(initial_count)

        return self._get_newest_symbol_id()

    def _open_symbols_tab(self) -> None:
        """Open the Symbols tab."""
        self._console.click("Symbols")

    def _count_symbols(self) -> int:
        """Count number of symbols on the schematic."""
        return len(self._page.locator("[data-testid^='rf__node-']").all())

    def _select_symbol_type(self, symbol_type: str) -> None:
        """Select a symbol type from the symbols panel."""
        if symbol_type == "Valve":
            self._console.click("Valves")
            self._console.click("Generic")
        else:
            search_input = self._page.locator(
                "div:has-text('Search Symbols') input[role='textbox']"
            ).first
            search_input.fill(symbol_type)
            self._console.click(symbol_type)

    def _wait_for_new_symbol(self, initial_count: int) -> None:
        """Wait for a new symbol to appear."""
        self._page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {initial_count}"
        )

    def _get_newest_symbol_id(self) -> str:
        """Get the ID of the newest symbol."""
        all_symbols = self._page.locator("[data-testid^='rf__node-']").all()
        return all_symbols[-1].get_attribute("data-testid") or "unknown"

    def valve(
        self,
        label: str,
        state_channel: str,
        command_channel: str,
        show_control_chip: bool | None = None,
        variant: Literal["valve", "threeway", "threeway_ball"] = "valve",
    ) -> Valve:
        """Create a valve symbol.

        Args:
            label: Display label for the symbol
            state_channel: Channel name for valve state
            command_channel: Channel name for valve commands
            show_control_chip: Whether to show the control chip (optional)
            variant: Type of valve ("valve", "threeway", "threeway_ball")

        Returns:
            Configured valve symbol instance
        """
        # Map variant to symbol type and class
        variant_map = {
            "valve": ("Valve", Valve),
            "threeway": ("Three Way", ValveThreeWay),
            "threeway_ball": ("Three-Way Ball", ValveThreeWayBall),
        }

        symbol_type, valve_class = variant_map[variant]
        symbol_id = self._add_symbol(symbol_type)

        valve = valve_class(self._page, self._console, symbol_id, label)
        valve.edit_properties(
            state_channel=state_channel,
            command_channel=command_channel,
            show_control_chip=show_control_chip,
        )
        return valve

    def setpoint(self, label: str, channel_name: str) -> Setpoint:
        """Create a setpoint symbol.

        Args:
            label: Display label for the symbol
            channel_name: Channel name for the setpoint

        Returns:
            Configured setpoint symbol instance
        """
        symbol_id = self._add_symbol("Setpoint")
        setpoint = Setpoint(self._page, self._console, symbol_id, label)
        setpoint.edit_properties(channel_name=channel_name)
        return setpoint

    def button(
        self,
        label: str,
        channel_name: str,
        activation_delay: float | None = None,
        show_control_chip: bool | None = None,
        mode: (
            Literal["fire", "momentary", "pulse", "Fire", "Momentary", "Pulse"] | None
        ) = None,
    ) -> Button:
        """Create a button symbol.

        Args:
            label: Display label for the symbol
            channel_name: Channel name for the button
            activation_delay: Delay before activation in seconds (optional)
            show_control_chip: Whether to show the control chip (optional)
            mode: Button mode - "fire", "momentary", or "pulse" (optional)

        Returns:
            Configured button symbol instance
        """
        symbol_id = self._add_symbol("Button")
        button = Button(self._page, self._console, symbol_id, label)
        button.edit_properties(
            channel_name=channel_name,
            activation_delay=activation_delay,
            show_control_chip=show_control_chip,
            mode=mode,
        )
        return button

    def value(
        self,
        label: str,
        channel_name: str,
        notation: str | None = None,
        precision: int | None = None,
        averaging_window: int | None = None,
        stale_color: str | None = None,
        stale_timeout: int | None = None,
    ) -> Value:
        """Create a value display symbol.

        Args:
            label: Display label for the symbol
            channel_name: Channel name for the value display
            notation: Number notation format (optional)
            precision: Decimal precision (optional)
            averaging_window: Averaging window size (optional)
            stale_color: Color for stale data (optional)
            stale_timeout: Timeout for stale data in milliseconds (optional)

        Returns:
            Configured value symbol instance
        """
        symbol_id = self._add_symbol("Value")
        value = Value(self._page, self._console, symbol_id, label)
        value.edit_properties(
            channel_name=channel_name,
            notation=notation,
            precision=precision,
            averaging_window=averaging_window,
            stale_color=stale_color,
            stale_timeout=stale_timeout,
        )
        return value
