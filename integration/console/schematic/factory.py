#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, TypeVar

from playwright.sync_api import Page

from console.console import Console

from .button import Button
from .setpoint import Setpoint
from .value import Value
from .valve import Valve
from .valve_threeway import ValveThreeWay
from .valve_threeway_ball import ValveThreeWayBall

ValveT = TypeVar("ValveT", bound=Valve)


class SchematicSymbolFactory:
    """Factory for creating and configuring schematic symbols.

    Handles both UI manipulation (adding symbols to schematic) and
    symbol instantiation/configuration. This eliminates the need for
    a separate coordinator layer.
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

    def setpoint(self, channel_name: str) -> Setpoint:
        """Create and configure a setpoint symbol.

        Args:
            channel_name: Channel name for the setpoint

        Returns:
            Configured Setpoint instance
        """
        symbol_id = self._add_symbol("Setpoint")
        setpoint = Setpoint(self._page, self._console, symbol_id, channel_name)
        setpoint.edit_properties(channel_name=channel_name)
        return setpoint

    def button(
        self,
        channel_name: str,
        activation_delay: float | None = None,
        show_control_chip: bool | None = None,
        mode: (
            Literal["fire", "momentary", "pulse", "Fire", "Momentary", "Pulse"] | None
        ) = None,
    ) -> Button:
        """Create and configure a button symbol.

        Args:
            channel_name: Channel name for the button
            activation_delay: Delay before activation in seconds
            show_control_chip: Whether to show the control chip
            mode: Button mode (fire, momentary, or pulse)

        Returns:
            Configured Button instance
        """
        symbol_id = self._add_symbol("Button")
        button = Button(self._page, self._console, symbol_id, channel_name)
        button.edit_properties(
            channel_name=channel_name,
            activation_delay=activation_delay,
            show_control_chip=show_control_chip,
            mode=mode,
        )
        return button

    def _create_valve_generic(
        self,
        symbol_type: str,
        valve_class: type[ValveT],
        channel_name: str,
        show_control_chip: bool | None = None,
        no_state_channel: bool = True,
    ) -> ValveT:
        """Generic valve creation logic shared by all valve types.

        Args:
            symbol_type: Symbol type string for _add_symbol() (e.g., "Valve", "Three Way")
            valve_class: Class to instantiate (Valve, ValveThreeWay, or ValveThreeWayBall)
            channel_name: Base channel name. Will be used for _state and _cmd channels unless no_state_channel=True.
            show_control_chip: Whether to show the control chip
            no_state_channel: If True, uses channel_name directly. If False, creates _state and _cmd channels.

        Returns:
            Configured valve instance
        """
        if not no_state_channel:
            ch_state = f"{channel_name}_state"
            ch_cmd = f"{channel_name}_cmd"
        else:
            ch_state = channel_name
            ch_cmd = channel_name

        symbol_id = self._add_symbol(symbol_type)
        valve = valve_class(self._page, self._console, symbol_id, channel_name)
        valve.edit_properties(
            state_channel=ch_state,
            command_channel=ch_cmd,
            show_control_chip=show_control_chip,
        )
        return valve

    def valve(
        self,
        channel_name: str,
        show_control_chip: bool | None = None,
        no_state_channel: bool = True,
    ) -> Valve:
        """Create and configure a generic valve symbol.

        Args:
            channel_name: Base channel name. Will be used for _state and _cmd channels unless no_state_channel=True.
            show_control_chip: Whether to show the control chip
            no_state_channel: If True, uses channel_name directly. If False, creates _state and _cmd channels.

        Returns:
            Configured Valve instance
        """
        return self._create_valve_generic(
            "Valve",
            Valve,
            channel_name,
            show_control_chip,
            no_state_channel,
        )

    def valve_threeway(
        self,
        channel_name: str,
        show_control_chip: bool | None = None,
        no_state_channel: bool = True,
    ) -> ValveThreeWay:
        """Create and configure a three-way valve symbol.

        Args:
            channel_name: Base channel name. Will be used for _state and _cmd channels unless no_state_channel=True.
            show_control_chip: Whether to show the control chip
            no_state_channel: If True, uses channel_name directly. If False, creates _state and _cmd channels.

        Returns:
            Configured ValveThreeWay instance
        """
        return self._create_valve_generic(
            "Three Way",
            ValveThreeWay,
            channel_name,
            show_control_chip,
            no_state_channel,
        )

    def valve_threeway_ball(
        self,
        channel_name: str,
        show_control_chip: bool | None = None,
        no_state_channel: bool = True,
    ) -> ValveThreeWayBall:
        """Create and configure a three-way ball valve symbol.

        Args:
            channel_name: Base channel name. Will be used for _state and _cmd channels unless no_state_channel=True.
            show_control_chip: Whether to show the control chip
            no_state_channel: If True, uses channel_name directly. If False, creates _state and _cmd channels.

        Returns:
            Configured ValveThreeWayBall instance
        """
        return self._create_valve_generic(
            "Three-Way Ball",
            ValveThreeWayBall,
            channel_name,
            show_control_chip,
            no_state_channel,
        )

    def value(
        self,
        channel_name: str,
        notation: str | None = None,
        precision: int | None = None,
        averaging_window: int | None = None,
        stale_color: str | None = None,
        stale_timeout: int | None = None,
    ) -> Value:
        """Create and configure a value symbol.

        Args:
            channel_name: Channel name for the value display
            notation: Number notation format
            precision: Decimal precision
            averaging_window: Averaging window size
            stale_color: Color for stale data
            stale_timeout: Timeout for stale data in milliseconds

        Returns:
            Configured Value instance
        """
        symbol_id = self._add_symbol("Value")
        value = Value(self._page, self._console, symbol_id, channel_name)

        value.edit_properties(
            channel_name=channel_name,
            notation=notation,
            precision=precision,
            averaging_window=averaging_window,
            stale_color=stale_color,
            stale_timeout=stale_timeout,
        )

        return value
