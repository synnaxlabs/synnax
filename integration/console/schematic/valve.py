#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import synnax as sy

from .symbol import Symbol


class Valve(Symbol):
    """Schematic valve symbol"""

    def __init__(
        self,
        *,
        label: str,
        state_channel: str,
        command_channel: str,
        show_control_chip: bool = True,
        symbol_type: str = "Valve",
        rotatable: bool = True,
    ):
        """Initialize a valve symbol with configuration.

        Args:
            label: Display label for the symbol
            state_channel: Channel name for valve state
            command_channel: Channel name for valve commands
            show_control_chip: Whether to show the control chip (optional)
            symbol_type: The type of symbol (default: "Valve")
            rotatable: Whether the symbol can be rotated (default: True)
        """
        super().__init__(label, symbol_type=symbol_type, rotatable=rotatable)
        self.state_channel = state_channel
        self.command_channel = command_channel
        self.show_control_chip = show_control_chip

    def _add_symbol_to_schematic(self, symbol_type: str) -> str:
        """Add a valve symbol using the Valves menu."""
        if self.page is None or self.console is None:
            raise RuntimeError("Symbol not attached to schematic")

        self.console.notifications.close_all()
        self.console.click("Symbols")
        initial_count = len(self.page.locator("[data-testid^='rf__node-']").all())

        self.console.click("Valves")
        self.console.click("Generic")

        self.page.wait_for_function(
            f"document.querySelectorAll('[data-testid^=\"rf__node-\"]').length > {initial_count}"
        )

        all_symbols = self.page.locator("[data-testid^='rf__node-']").all()
        return all_symbols[-1].get_attribute("data-testid") or "unknown"

    def _apply_properties(self) -> None:
        """Apply valve configuration after being added to schematic."""
        self.set_properties(
            state_channel=self.state_channel,
            command_channel=self.command_channel,
            show_control_chip=self.show_control_chip,
        )

    def set_properties(
        self,
        channel_name: str | None = None,
        state_channel: str | None = None,
        command_channel: str | None = None,
        show_control_chip: bool | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Set Valve properties including channel settings."""
        self.click()

        applied_properties: dict[str, Any] = {}

        # Navigate to Properties > Control tab
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Control").last.click()

        if state_channel is not None:
            self.set_channel("State Channel", state_channel)
            applied_properties["state_channel"] = state_channel

        if command_channel is not None:
            self.set_channel("Command Channel", command_channel)
            applied_properties["command_channel"] = command_channel

        if show_control_chip is not None:
            chip_toggle = (
                self.page.locator("text=Show Control Chip")
                .locator("..")
                .locator("input[type='checkbox']")
            )
            if chip_toggle.count() > 0:
                current_state = chip_toggle.is_checked()
                if current_state != show_control_chip:
                    chip_toggle.click()
            applied_properties["show_control_chip"] = show_control_chip

        return applied_properties

    def get_properties(self, tab: str = "Control") -> dict[str, Any]:
        """Get the current properties of the symbol"""
        super().get_properties(tab=tab)

        props: dict[str, Any] = {
            "channel": "",
            "activation_delay": -1.0,
            "show_control_chip": False,
            "mode": "",
        }

        # State Channel
        state_channel = (
            self.page.locator("text=State Channel").locator("..").locator("button")
        )
        if state_channel.count() > 0:
            props["state_channel"] = state_channel.inner_text().strip()

        # State Channel
        command_channel = (
            self.page.locator("text=Command Channel").locator("..").locator("button")
        )
        if command_channel.count() > 0:
            props["command_channel"] = command_channel.inner_text().strip()

        # Show Control Chip
        chip_toggle = (
            self.page.locator("text=Show Control Chip")
            .locator("..")
            .locator("input[type='checkbox']")
        )
        if chip_toggle.count() > 0:
            props["show_control_chip"] = chip_toggle.is_checked()

        return props

    def press(self, sleep: int = 100) -> None:
        """Press button

        Args:
            sleep: Time in milliseconds to wait after pressing. Buffer for network delays and slow animations.
        """

        self._disable_edit_mode()
        self.click(sleep=sleep)

    def press_and_hold(self, delay: sy.TimeSpan = sy.TimeSpan.SECOND) -> None:
        """Click and hold the button for the specified duration."""
        self._disable_edit_mode()
        self.page.mouse.down()
        self.page.wait_for_timeout(int(delay.milliseconds()))
        self.page.mouse.up()

    def open(self) -> None:
        """Press if closed, do nothing if open"""
        raise NotImplementedError("Not implemented")

    def close(self) -> None:
        """Press if open, do nothing if closed"""
        raise NotImplementedError("Not implemented")
