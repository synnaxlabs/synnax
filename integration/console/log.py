#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from synnax.channel.payload import ChannelName

from .console import Console
from .page import ConsolePage


class Log(ConsolePage):
    """Log page management interface"""

    page_type: str = "Log"
    pluto_label: str = ".pluto-log"

    @classmethod
    def open_from_search(cls, console: Console, name: str) -> Log:
        """Open an existing log by searching its name in the command palette.

        Args:
            console: Console instance.
            name: Name of the log to search for and open.

        Returns:
            Log instance for the opened log.
        """
        console.search_palette(name)

        log_pane = console.page.locator(cls.pluto_label)
        log_pane.first.wait_for(state="visible", timeout=5000)

        log = cls(console, name, _skip_create=True)
        log.pane_locator = log_pane.first
        return log

    def __init__(
        self,
        console: Console,
        page_name: str,
        channel_name: ChannelName | None = None,
        *,
        _skip_create: bool = False,
    ) -> None:
        """
        Initialize a Log page.

        Args:
            console: Console instance
            page_name: Name for the page
            channel_name: Optional channel to set for the log page
            _skip_create: Internal flag to skip page creation (used by factory methods)
        """
        super().__init__(console, page_name, _skip_create=_skip_create)

        if channel_name is not None:
            self.set_channel(channel_name)

    def set_channel(self, channel_name: str) -> None:
        self.console.click_btn("Channel")
        self.console.select_from_dropdown(channel_name, "Select a Channel")

    def has_channel(self, channel_name: str) -> bool:
        """Check if a channel is shown in the Log toolbar."""
        self.console.layout.get_tab(self.page_name).click()
        self.console.layout.show_visualization_toolbar()
        channel_btn = (
            self.page.locator("text=Channel").locator("..").locator("button").first
        )
        channel_text = channel_btn.inner_text().strip()
        result = channel_name in channel_text
        return result

    def copy_link(self) -> str:
        """Copy link to the log via the toolbar link button."""
        self.console.notifications.close_all()
        self.console.layout.show_visualization_toolbar()
        link_button = self.page.locator(".pluto-icon--link").locator("..")
        link_button.click(timeout=5000)

        try:
            link: str = str(self.page.evaluate("navigator.clipboard.readText()"))
            return link
        except Exception:
            return ""

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

    def wait_until_streaming(self, timeout_ms: int = 5000) -> bool:
        """Wait until the log starts streaming data.

        Args:
            timeout_ms: Maximum time to wait in milliseconds.

        Returns:
            True if streaming started, False if timeout reached.

        Raises:
            Exception: Re-raises any non-timeout exceptions.
        """
        live_button = self.page.locator(
            f"{self.pluto_label} button.pluto-log__live"
        ).first
        try:
            live_button.wait_for(state="visible", timeout=timeout_ms)
            return True
        except Exception as e:
            if "Timeout" in type(e).__name__:
                return False
            raise RuntimeError from e

    def wait_until_waiting_for_data(self, timeout_ms: int = 5000) -> bool:
        """Wait until the log shows 'No data received yet' message.

        Args:
            timeout_ms: Maximum time to wait in milliseconds.

        Returns:
            True if waiting state reached, False if timeout reached.

        Raises:
            Exception: Re-raises any non-timeout exceptions.
        """
        waiting_message = self.page.locator(self.pluto_label).get_by_text(
            "No data received yet"
        )
        try:
            waiting_message.wait_for(state="visible", timeout=timeout_ms)
            return True
        except Exception as e:
            if "Timeout" in type(e).__name__:
                return False
            raise RuntimeError from e

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
