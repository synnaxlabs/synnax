#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Custom symbol class for user-created symbols."""

from typing import Any

from console.schematic.symbol import Symbol


class CustomSymbol(Symbol):
    """Symbol subclass for user-created custom symbols.

    Custom symbols are created via the Symbol Editor (SVG upload) and stored
    in user-created groups. Unlike built-in symbols, they use the SymbolToolbar
    to select from a specific group rather than searching.

    Custom symbols can be configured as actuators with command channels.
    """

    def __init__(
        self,
        *,
        label: str,
        symbol_name: str,
        group_name: str,
        command_channel: str | None = None,
    ):
        """Initialize a custom symbol with configuration.

        Args:
            label: Display label for the symbol instance
            symbol_name: Name of the custom symbol in the library
            group_name: Name of the user-created group containing this symbol
            command_channel: Optional command channel name for actuator control
                           (also used as state channel)
        """
        super().__init__(label, symbol_type=symbol_name, rotatable=False)
        self._symbol_group = group_name
        self.command_channel = command_channel

    def _apply_properties(self) -> None:
        """Apply actuator configuration if command_channel is set."""
        if self.command_channel is not None:
            self.set_properties(channel_name=self.command_channel)

    def set_label(self, label: str) -> None:
        """Custom symbols don't have a standard label field in properties."""
        self.label = label

    def set_properties(
        self,
        channel_name: str | None = None,
        **kwargs: str,
    ) -> dict[str, str]:
        """Set custom symbol properties including command channel for actuator control.

        Args:
            channel_name: Command channel name for actuator control (also used as state)
            **kwargs: Additional properties (currently unused)

        Returns:
            Dictionary of applied properties
        """
        applied_properties: dict[str, str] = {}

        self.locator.wait_for(state="visible", timeout=2000)

        inner_div = self.locator.locator("div").first
        inner_div.click()
        self.page.locator(".react-flow__node.selected").wait_for(
            state="visible", timeout=2000
        )
        self.layout.show_visualization_toolbar()

        self.page.get_by_text("Properties").click()
        control_tab = self.page.locator("#control").nth(1)
        control_tab.wait_for(state="visible", timeout=5000)
        control_tab.click()

        if channel_name is not None:
            self.set_channel(input_field="Command Channel", channel_name=channel_name)
            applied_properties["command_channel"] = channel_name
            self.set_channel(input_field="State Channel", channel_name=channel_name)
            applied_properties["state_channel"] = channel_name

        return applied_properties

    def press(self) -> None:
        """Press the custom symbol to actuate it."""
        self._disable_edit_mode()
        self.click()
