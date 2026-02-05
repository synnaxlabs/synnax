#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING

from playwright.sync_api import FrameLocator, Locator, Page

if TYPE_CHECKING:
    from .console import Console


class DocsClient:
    """Documentation client for Console UI automation."""

    def __init__(self, page: Page, console: "Console"):
        self.page = page
        self.console = console

    def open_via_command_palette(self) -> None:
        self.console.command_palette("Read the documentation")
        self._wait_for_docs_tab()

    def open_via_question_mark_icon(self) -> None:
        btn = self.page.locator(".console-docs__open-button")
        btn.wait_for(state="visible", timeout=5000)
        btn.click()
        self._wait_for_docs_tab()

    def close(self) -> None:
        self.console.layout.close_tab("Documentation")

    @property
    def is_open(self) -> bool:
        return self.console.layout.get_tab("Documentation").count() > 0

    def _get_iframe(self) -> Locator:
        return self.page.locator(".console-docs iframe")

    def get_frame(self) -> FrameLocator:
        return self.page.frame_locator(".console-docs iframe")

    def get_iframe_url(self) -> str:
        iframe = self._get_iframe()
        iframe.wait_for(state="visible", timeout=10000)
        return iframe.get_attribute("src") or ""

    def has_text(self, text: str, timeout: int = 10000) -> bool:
        try:
            self.get_frame().get_by_text(text).first.wait_for(
                state="visible", timeout=timeout
            )
            return True
        except Exception:
            return False

    def wait_for_iframe_loaded(self, timeout: int = 15000) -> None:
        self._get_iframe().wait_for(state="visible", timeout=timeout)
        loader = self.page.locator(".console-docs .pluto--loader")
        if loader.count() > 0:
            try:
                loader.wait_for(state="hidden", timeout=timeout)
            except Exception:
                pass

    def _wait_for_docs_tab(self) -> None:
        self.console.layout.get_tab("Documentation").wait_for(
            state="visible", timeout=10000
        )
