#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

import synnax as sy
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
        channel_name: str | None = None,
        *,
        pane_locator: Locator,
    ) -> None:
        """Initialize a Log page wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(layout, client, page_name, pane_locator=pane_locator)

        if channel_name is not None:
            self.set_channel(channel_name)

    def clear_channels(self) -> None:
        """Remove all currently selected channels from the log."""
        self.layout.show_visualization_toolbar()
        toolbar = self.page.locator(".console-log-toolbar")
        while True:
            remove_btns = toolbar.locator(
                ".console-log__channel-row .pluto-btn--text:not(.pluto--disabled)"
            )
            if remove_btns.count() == 0:
                break
            remove_btns.first.click()

    def set_channel(self, channel_name: str) -> None:
        """Add a channel to the log via the 'Add a channel...' row."""
        self.layout.show_visualization_toolbar()
        toolbar = self.page.locator(".console-log-toolbar")
        add_trigger = toolbar.get_by_text("Add a channel...")
        add_trigger.click()
        self.layout.select_from_dropdown(channel_name, "Search channels")

    def has_channel(self, channel_name: str) -> bool:
        """Check if a channel is shown in the Log toolbar."""
        self.layout.get_tab(self.page_name).click()
        self.layout.show_visualization_toolbar()
        toolbar = self.page.locator(".console-log-toolbar")
        rows = toolbar.locator(".console-log__channel-row")
        for i in range(rows.count()):
            row_text = rows.nth(i).inner_text() or ""
            if channel_name in row_text:
                return True
        return False

    def copy_all_entries(self) -> str:
        """Select all rendered log entries and copy them to the clipboard.

        Log entries are drawn on a canvas, so they are not countable as DOM
        elements. Select-all (Ctrl+A) followed by copy (Ctrl+C) round-trips the
        rendered entries through the clipboard, which is the only fidelity-true
        proxy for what the user actually sees.

        Returns:
            The copied entry text (one entry per line).
        """
        self.layout.get_tab(self.page_name).click()
        assert self.pane_locator is not None, "Log pane should be visible"
        self.pane_locator.click(position={"x": 40, "y": 200})
        self.page.keyboard.press("Control+a")
        self.page.keyboard.press("Control+c")
        return self.layout.read_clipboard()

    def entry_count(self) -> int:
        """Return the number of rendered log entries via select-all + copy."""
        text = self.copy_all_entries()
        return len([line for line in text.split("\n") if line != ""])

    def wait_for_entry_count(self, expected: int, retries: int = 20) -> int:
        """Poll until at least ``expected`` entries are rendered, returning the
        final observed count (which may be less than ``expected`` on timeout).
        """
        count = 0
        for _ in range(retries):
            count = self.entry_count()
            if count >= expected:
                return count
            self.page.wait_for_timeout(250)
        return count

    def is_empty(self) -> bool:
        """Check if the log shows any empty state message."""
        if not self.pane_locator:
            return True
        no_channel = (
            self.pane_locator.locator("text=No channels configured").count() > 0
        )
        no_data = self.pane_locator.locator("text=No data received yet").count() > 0
        return no_channel or no_data

    def needs_channel_configured(self) -> bool:
        """Check if the log shows 'No channel configured' message."""
        if not self.pane_locator:
            return False
        return self.pane_locator.locator("text=No channels configured").count() > 0

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
