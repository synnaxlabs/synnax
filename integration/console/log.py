#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.channel.payload import ChannelName

from .console import Console
from .page import ConsolePage


class Log(ConsolePage):
    """Log page management interface"""

    page_type: str = "Log"
    pluto_label: str = ".pluto-log"

    def __init__(
        self,
        client: sy.Synnax,
        console: Console,
        page_name: str,
        channel_name: ChannelName | None = None,
    ) -> None:
        """
        Initialize a Log page.

        Args:
            client: Synnax client instance
            console: Console instance
            page_name: Name for the page
            channel_name: Optional channel to set for the log page
        """
        super().__init__(client, console, page_name)

        if channel_name is not None:
            self.set_channel(channel_name)

    def set_channel(self, channel_name: str) -> None:
        self.console.click_btn("Channel")
        self.console.select_from_dropdown(channel_name, "Select a Channel")
