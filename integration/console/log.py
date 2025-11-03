#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING

from playwright.sync_api import Page
from synnax.channel.payload import (
    ChannelName,
)

from .page import ConsolePage

if TYPE_CHECKING:
    from .console import Console


class Log(ConsolePage):
    """Log page management interface"""

    channel_name: ChannelName | None

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "Log"
        self.pluto_label = ".pluto-log"

    def new(self, channel_name: ChannelName | None = None) -> str:
        page_id = super().new()
        if channel_name is not None:
            self.set_channel(channel_name)
        return page_id

    def set_channel(self, channel_name: str) -> None:
        self.console.click_btn("Channel")
        self.console.select_from_dropdown(channel_name, "Select a Channel")
