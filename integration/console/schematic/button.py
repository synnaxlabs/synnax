#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal

import synnax as sy

from .symbol import Symbol

ButtonMode = Literal["Fire", "Momentary", "Pulse"]


class Button(Symbol):
    """Schematic button symbol"""

    def __init__(
        self,
        *,
        label: str,
        channel_name: str,
        activation_delay: float | None = None,
        show_control_chip: bool | None = None,
        mode: ButtonMode = "Fire",
        symbol_type: str = "Button",
    ):
        """Initialize a button symbol with configuration.

        Args:
            label: Display label for the symbol
            channel_name: Channel name for the button
            activation_delay: Delay before activation in seconds (optional)
            show_control_chip: Whether to show the control chip (optional)
            mode: Button mode - "Fire", "Momentary", or "Pulse" (optional)
            symbol_type: The type of symbol (default: "Button")
        """
        super().__init__(label, symbol_type=symbol_type, rotatable=False)
        self.channel_name = channel_name
        self.activation_delay = activation_delay
        self.show_control_chip = show_control_chip
        self.mode = mode

    def _apply_properties(self) -> None:
        """Apply button configuration after being added to schematic."""
        self.set_properties(
            channel_name=self.channel_name,
            activation_delay=self.activation_delay,
            show_control_chip=self.show_control_chip,
            mode=self.mode,
        )

    def set_properties(
        self,
        channel_name: str | None = None,
        activation_delay: float | None = None,
        show_control_chip: bool | None = None,
        mode: ButtonMode = "Fire",
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Set Button properties including channel settings."""
        self.click()

        applied_properties: dict[str, Any] = {}
        if channel_name is not None:
            self.set_label(channel_name)

        # Navigate to Properties > Control tab
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Control").last.click()

        if channel_name is not None:
            self.set_channel("Output Channel", channel_name)
            applied_properties["channel"] = channel_name

        if activation_delay is not None:
            self.console.fill_input_field("Activation Delay", str(activation_delay))
            self.page.keyboard.press("Enter")
            applied_properties["activation_delay"] = activation_delay

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

        if mode is not None:
            self.page.get_by_text(mode).click()
            applied_properties["mode"] = mode

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

        # Channel Name
        channel_display = (
            self.page.locator("text=Output Channel").locator("..").locator("button")
        )
        if channel_display.count() > 0:
            props["channel"] = channel_display.inner_text().strip()

        # Activation Delay
        props["activation_delay"] = float(
            self.console.get_input_field("Activation Delay")
        )

        # Show Control Chip
        chip_toggle = (
            self.page.locator("text=Show Control Chip")
            .locator("..")
            .locator("input[type='checkbox']")
        )
        if chip_toggle.count() > 0:
            props["show_control_chip"] = chip_toggle.is_checked()

        # Mode
        mode_options = ["Fire", "Momentary", "Pulse"]
        for option in mode_options:
            try:
                button = self.page.get_by_text(option).first
                if button.count() > 0:
                    class_name = button.get_attribute("class") or ""
                    if "pluto-btn--filled" in class_name:
                        props["mode"] = option
                        break
            except Exception as e:
                raise RuntimeError(f"Error getting mode property: {e}")

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
