#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import FrameLocator, Locator
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from .base import BaseClient
from .layout import LayoutClient


class DocsClient(BaseClient):
    """Documentation client for Console UI automation."""

    def __init__(self, layout: LayoutClient):
        super().__init__(layout)

    def open_via_command_palette(self) -> None:
        """Open the documentation page via the command palette."""
        self.layout.command_palette("Read the documentation")
        self._wait_for_docs_tab()

    def open_via_question_mark_icon(self) -> None:
        """Open the documentation page by clicking the question mark icon."""
        btn = self.layout.page.locator(".console-docs__open-button")
        btn.wait_for(state="visible", timeout=5000)
        btn.click()
        self._wait_for_docs_tab()

    def close(self) -> None:
        """Close the documentation tab."""
        self.layout.close_tab("Documentation")

    @property
    def is_open(self) -> bool:
        """Check if the documentation tab is currently open."""
        return self.layout.get_tab("Documentation").count() > 0

    def _get_iframe(self) -> Locator:
        """Get the documentation iframe element locator."""
        return self.layout.page.locator(".console-docs iframe")

    def get_frame(self) -> FrameLocator:
        """Get the documentation iframe as a FrameLocator for content interaction."""
        return self.layout.page.frame_locator(".console-docs iframe")

    def get_iframe_url(self) -> str:
        """Get the current URL loaded in the documentation iframe.

        Returns:
            The iframe src attribute value, or empty string if not set.
        """
        iframe = self._get_iframe()
        iframe.wait_for(state="visible", timeout=10000)
        return iframe.get_attribute("src") or ""

    def has_text(self, text: str) -> bool:
        """Check if the documentation iframe contains specific text."""
        try:
            self.get_frame().get_by_text(text).first.wait_for(
                state="visible", timeout=5000
            )
            return True
        except PlaywrightTimeoutError:
            return False

    def wait_for_iframe_loaded(self) -> None:
        """Wait for the documentation iframe to be fully loaded."""
        self._get_iframe().wait_for(state="visible", timeout=5000)
        loader = self.layout.page.locator(".console-docs .pluto--loader")
        if loader.count() > 0:
            try:
                loader.wait_for(state="hidden", timeout=5000)
            except PlaywrightTimeoutError:
                pass

    def _wait_for_docs_tab(self) -> None:
        """Wait for the Documentation tab to become visible."""
        self.layout.get_tab("Documentation").wait_for(state="visible", timeout=10000)
