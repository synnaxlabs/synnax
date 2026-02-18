#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from console.schematic.symbol import Symbol


class Select(Symbol):
    """Schematic select symbol (write-only dropdown + send button)"""

    _symbol_group = "General"

    def __init__(
        self,
        *,
        label: str,
        channel_name: str,
        options: list[dict[str, Any]] | None = None,
    ):
        """Initialize a select symbol with configuration.

        Args:
            label: Display label for the symbol
            channel_name: Channel name for the select output
            options: List of option dicts with "name" and "value" keys
        """
        super().__init__(label, symbol_type="Select", rotatable=False)
        self.channel_name = channel_name
        self.options = options or []

    def _apply_properties(self) -> None:
        self.set_properties(channel_name=self.channel_name, options=self.options)

    def set_properties(
        self,
        channel_name: str | None = None,
        options: list[dict[str, Any]] | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        self.click()
        applied_properties: dict[str, Any] = {}

        if channel_name is not None:
            self.set_label(channel_name)
            self.page.get_by_text("Properties").click()
            self.page.get_by_text("Control").last.click()
            self.set_channel(input_field="Command Channel", channel_name=channel_name)
            applied_properties["channel"] = channel_name

        if options is not None and len(options) > 0:
            self.click()
            self.page.get_by_text("Properties").click()
            self.page.get_by_text("Options", exact=True).click()
            for i, option in enumerate(options):
                if i == 0:
                    self.page.get_by_text("Add an option").click()
                else:
                    self.page.get_by_text("Add option", exact=True).click()
                items = self.page.locator(".pluto-list__item").all()
                last_item = items[-1]
                name_input = last_item.locator("input").first
                name_input.fill(option["name"])
                value_input = last_item.locator("input").nth(1)
                value_input.fill(str(option["value"]))
            applied_properties["options"] = options

        return applied_properties

    def send(self, option_name: str) -> None:
        """Select an option from the dropdown and click Send.

        Args:
            option_name: The display name of the option to select
        """
        self._disable_edit_mode()
        self.notifications.close_all()
        self.page.wait_for_timeout(300)

        trigger = self.locator.locator(".pluto-dialog__trigger").first
        trigger.click()
        self.layout.select_from_dropdown(option_name)

        send_button = self.locator.locator("button").filter(has_text="Send")
        send_button.click(timeout=500)
