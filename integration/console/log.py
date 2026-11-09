#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from collections.abc import Callable

from playwright.sync_api import Locator, TimeoutError as PlaywrightTimeoutError, expect

import synnax as sy
from console.layout import LayoutClient
from console.page import ConsolePage, PageType


class Log(ConsolePage):
    """Log page management interface"""

    page_type: PageType = "log"
    pluto_label: str = "role=log"

    def __init__(
        self,
        *,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        channel_name: str | None = None,
        pane_locator: Locator,
    ) -> None:
        """Initialize a Log page wrapper (see ConsolePage.__init__ for details)."""
        super().__init__(
            layout=layout, client=client, page_name=page_name, pane_locator=pane_locator
        )

        if channel_name is not None:
            self.set_channel(channel_name)

    def _pane(self) -> Locator:
        if self.pane_locator is None:
            raise RuntimeError("Log pane locator not available")
        return self.pane_locator

    def _channels(self) -> Locator:
        """The list of the log's channel rows in the Log toolbar."""
        self.layout.show_visualization_toolbar()
        return self.page.get_by_role("list", name="Log channels")

    def clear_channels(self) -> None:
        """Remove all currently selected channels from the log.

        Clearing a row's channel select removes the row. The clear button
        reveals on hover over the select, so dispatch the click on it instead of
        moving the mouse there.
        """
        rows = self._channels().get_by_role("listitem")
        while (count := rows.count()) > 0:
            rows.first.get_by_role(
                "button", name=re.compile(r"^Clear ")
            ).dispatch_event("click")
            expect(rows).to_have_count(count - 1, timeout=2000)

    def set_channel(self, channel_name: str) -> None:
        """Add a channel to the log via the 'Add channel' row."""
        self._channels().get_by_text("Add channel").click()
        self.layout.select_from_dropdown(channel_name, placeholder="Search channels")

    def has_channel(self, channel_name: str) -> bool:
        """Check if a channel is shown in the Log toolbar."""
        self.layout.get_tab(self.page_name).click()
        clear = self._channels().get_by_role(
            "button", name=f"Clear {channel_name}", exact=True
        )
        return clear.count() > 0

    def copy_visible_entries(self) -> str:
        """Drag-select the log and copy the selection via the context menu.

        Log entries are drawn on a canvas, so they are not countable as DOM
        nodes. A mouse drag (not the Ctrl+A trigger) sets the selection, and the
        context-menu Copy item writes it through ``navigator.clipboard.write``
        (the async clipboard API). Both avoid the focus-dependent native copy
        event and the active-tab trigger gate, so the round-trip is reliable in
        headless runs. Returns "" if the copy could not be performed.
        """
        self.layout.get_tab(self.page_name).click()
        pane = self._pane()
        bbox = pane.bounding_box()
        if bbox is None:
            return ""
        x = bbox["x"] + 40
        self.page.mouse.move(x, bbox["y"] + 10)
        self.page.mouse.down()
        self.page.mouse.move(x, bbox["y"] + bbox["height"] - 10, steps=5)
        self.page.mouse.up()
        pane.click(button="right", position={"x": 40, "y": 10})
        copy = self.page.get_by_role("menu").get_by_role(
            "menuitem", name="Copy", exact=True
        )
        try:
            copy.click(timeout=3000)
        except PlaywrightTimeoutError:
            self.page.keyboard.press("Escape")
            return ""
        return self.layout.read_clipboard()

    def wait_for_copied_entries(
        self,
        *,
        retries: int = 20,
        until: Callable[[list[str]], bool] = bool,
    ) -> list[str]:
        """Poll the drag-select + context-menu copy until the copied lines pass
        ``until``, which by default accepts any non-empty copy.

        Returns the non-empty copied lines (one per rendered entry) of the last
        attempt, whether or not they passed ``until``.
        """
        lines: list[str] = []
        for _ in range(retries):
            text = self.copy_visible_entries()
            lines = [ln for ln in text.split("\n") if ln.strip()]
            if until(lines):
                return lines
            self.page.wait_for_timeout(250)
        return lines

    def scroll_back(self, pixels: int) -> None:
        """Scroll the log up by ``pixels`` with the mouse wheel."""
        box = self._pane().bounding_box()
        if box is None:
            raise RuntimeError("Log pane has no bounding box")
        self.page.mouse.move(box["x"] + box["width"] / 2, box["y"] + box["height"] / 2)
        self.page.mouse.wheel(0, -pixels)

    def is_empty(self) -> bool:
        """Check if the log shows any empty state message."""
        return self.needs_channel_configured() or self.is_waiting_for_data()

    def needs_channel_configured(self) -> bool:
        """Check if the log shows 'No channels configured' message."""
        return self._pane().get_by_text("No channels configured").is_visible()

    def is_waiting_for_data(self) -> bool:
        """Check if the log shows 'No data received yet' message."""
        return self._pane().get_by_text("No data received yet").is_visible()

    def _hold_toggle(self) -> Locator:
        """The pause/resume scrolling toggle in the log's hover controls."""
        return self._pane().get_by_role("button", name="Pause scrolling", exact=True)

    def wait_until_streaming(self) -> bool:
        """Wait until streamed entries replace the log's empty state."""
        empty_state = self._pane().get_by_text("No data received yet")
        try:
            empty_state.wait_for(state="hidden", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_until_waiting_for_data(self) -> bool:
        """Wait until the log shows 'No data received yet' message."""
        waiting_message = self._pane().get_by_text("No data received yet")
        try:
            waiting_message.wait_for(state="visible", timeout=5000)
            return True
        except PlaywrightTimeoutError:
            return False

    def is_scrolling_paused(self) -> bool:
        """Check if log scrolling is paused (hold toggle pressed)."""
        return self._hold_toggle().get_attribute("aria-pressed") == "true"

    def wait_until_scrolling_paused(self) -> bool:
        """Wait until log scrolling is paused."""
        try:
            expect(self._hold_toggle()).to_have_attribute("aria-pressed", "true")
            return True
        except AssertionError:
            return False

    def pause_scrolling(self) -> None:
        """Pause log scrolling (enter scrollback mode)."""
        self._set_hold(True)

    def resume_scrolling(self) -> None:
        """Resume log scrolling (exit scrollback mode)."""
        self._set_hold(False)

    def _set_hold(self, hold: bool) -> None:
        if self.is_scrolling_paused() == hold:
            return
        # The controls reveal on pane hover; hover first so the click lands.
        self._pane().hover()
        toggle = self._hold_toggle()
        toggle.click()
        expect(toggle).to_have_attribute("aria-pressed", str(hold).lower())
