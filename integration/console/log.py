#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError
from synnax.channel.payload import ChannelName

from console.layout import LayoutClient
from console.page import ConsolePage


class Log(ConsolePage):
    """Log page management interface"""

    page_type: str = "Log"
    pluto_label: str = ".pluto-log"

    def __init__(
        self,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        channel_name: ChannelName | None = None,
        *,
        pane_locator: Locator,
    ) -> None:
        """Initialize a Log page wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(layout, client, page_name, pane_locator=pane_locator)

        if channel_name is not None:
            self.set_channel(channel_name)

    def set_channel(self, channel_name: str) -> None:
        self.layout.show_visualization_toolbar()
        self.layout.click_btn("Channel")
        self.layout.select_from_dropdown(channel_name, "Select a Channel")

    def has_channel(self, channel_name: str) -> bool:
        """Check if a channel is shown in the Log toolbar."""
        self.layout.get_tab(self.page_name).click()
        self.layout.show_visualization_toolbar()
        channel_btn = (
            self.page.locator("text=Channel").locator("..").locator("button").first
        )
        channel_text = channel_btn.inner_text().strip()
        result = channel_name in channel_text
        return result

    def is_empty(self) -> bool:
        """Check if the log shows any empty state message."""
        if not self.pane_locator:
            return True
        no_channel = self.pane_locator.locator("text=No channel configured").count() > 0
        no_data = self.pane_locator.locator("text=No data received yet").count() > 0
        return no_channel or no_data

    def needs_channel_configured(self) -> bool:
        """Check if the log shows 'No channel configured' message."""
        if not self.pane_locator:
            return False
        return self.pane_locator.locator("text=No channel configured").count() > 0

    def is_waiting_for_data(self) -> bool:
        """Check if the log shows 'No data received yet' message."""
        if not self.pane_locator:
            return False
        return self.pane_locator.locator("text=No data received yet").count() > 0

    def is_streaming(self) -> bool:
        """Check if the log is actively streaming data (live button visible)."""
        if not self.pane_locator:
            return False
        live_button = self.pane_locator.locator("button.pluto-log__live")
        return live_button.count() > 0

    def wait_until_streaming(self) -> bool:
        """Wait until the log starts streaming data."""
        live_button = self.page.locator(
            f"{self.pluto_label} button.pluto-log__live"
        ).first
        try:
            live_button.wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_until_waiting_for_data(self) -> bool:
        """Wait until the log shows 'No data received yet' message."""
        waiting_message = self.page.locator(self.pluto_label).get_by_text(
            "No data received yet"
        )
        try:
            waiting_message.wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

    def is_scrolling_paused(self) -> bool:
        """Check if log scrolling is paused."""
        if not self.pane_locator:
            return False
        live_button = self.pane_locator.locator("button.pluto-log__live")
        if live_button.count() == 0:
            return False
        btn_class = live_button.get_attribute("class") or ""
        return "pluto--active" in btn_class

    def pause_scrolling(self) -> None:
        """Pause log scrolling (enter scrollback mode)."""
        if self.is_scrolling_paused():
            return
        if not self.pane_locator:
            return
        live_button = self.pane_locator.locator("button.pluto-log__live")
        if live_button.count() > 0:
            live_button.click()

    def resume_scrolling(self) -> None:
        """Resume log scrolling (exit scrollback mode)."""
        if not self.is_scrolling_paused():
            return
        if not self.pane_locator:
            return
        live_button = self.pane_locator.locator("button.pluto-log__live")
        if live_button.count() > 0:
            live_button.click()
