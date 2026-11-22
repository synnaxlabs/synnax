#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.notifications import click_past
from console.schematic.symbol import Symbol


class Light(Symbol):
    """Schematic light symbol (read-only boolean display)"""

    _symbol_group: str | None = "General"

    def __init__(self, *, label: str, channel_name: str):
        """Initialize a light symbol with configuration.

        :param label: Display label for the symbol
        :param channel_name: Channel name for the light state
        """
        super().__init__(label=label, symbol_type="Light", rotatable=True)
        self.channel_name = channel_name

    def _apply_properties(self) -> None:
        self.click()
        self.open_properties_tab()
        click_past(self.page.get_by_text("Telemetry"))
        self.set_channel(section="State", channel_name=self.channel_name)
