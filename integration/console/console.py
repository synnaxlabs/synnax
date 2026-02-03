#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from playwright.sync_api import Locator, Page
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError

from framework.utils import get_results_path

from .access import AccessClient
from .arc import ArcClient
from .channels import ChannelClient
from .docs import DocsClient
from .labels import LabelClient
from .layout import LayoutClient
from .notifications import NotificationsClient
from .rack import RackClient
from .ranges import RangesClient
from .workspace import WorkspaceClient


class Console:
    """
    Console UI automation interface - thin orchestration layer.

    Composes specialized clients for interacting with the Synnax Console application.
    Layout operations are accessed through self.layout (LayoutClient).
    Each client imports and instantiates its own utilities (notifications, ctx_menu, tree).
    """

    access: AccessClient
    arc: ArcClient
    channels: ChannelClient
    client: sy.Synnax
    docs: DocsClient
    labels: LabelClient
    layout: LayoutClient
    notifications: NotificationsClient
    rack: RackClient
    ranges: RangesClient
    workspace: WorkspaceClient
    page: Page

    def __init__(self, page: Page, client: sy.Synnax):
        self.page = page
        self.client = client
        self.layout = LayoutClient(page)
        self.notifications = NotificationsClient(page)
        self.docs = DocsClient(self.layout)
        self.labels = LabelClient(self.layout)
        self.rack = RackClient(self.layout)
        self.arc = ArcClient(self.layout)
        self.access = AccessClient(self.layout)
        self.channels = ChannelClient(self.layout, self.client)
        self.ranges = RangesClient(self.layout)
        self.workspace = WorkspaceClient(self.layout, self.client)

    def check_for_error_screen(self) -> None:
        """Checks for 'Something went wrong' text and clicks 'Try again' if found"""
        sy.sleep(0.3)
        if self.page.get_by_text("Something went wrong").is_visible():
            sy.sleep(0.2)
            self.page.get_by_text("Try again").click()
            sy.sleep(0.2)

    def screenshot(self, name: str | None = None) -> None:
        """Take a screenshot of the entire console page."""
        if name is None:
            name = "console.png"
        elif not name.endswith(".png"):
            name = name + ".png"

        self.page.screenshot(
            path=get_results_path(name),
            full_page=True,
            animations="disabled",
            type="png",
        )

    def reload(self) -> None:
        """Reload the console page."""
        self.page.reload()
        self.page.wait_for_load_state("load", timeout=30000)
        self.page.wait_for_load_state("networkidle", timeout=30000)

    def _dismiss_unsaved_changes_dialog(self) -> None:
        """Dismiss the 'Lose Unsaved Changes' dialog if present."""
        if self.page.get_by_text("Lose Unsaved Changes").count() > 0:
            self.page.get_by_role("button", name="Confirm").click()

    def _find_tab_to_close(self, except_tabs: list[str]) -> Locator | None:
        """Find the first tab that should be closed.

        Skips tabs that become stale during iteration (can happen if DOM updates).
        """
        for tab in self.page.locator(".pluto-tabs-selector__btn").all():
            try:
                name = tab.inner_text(timeout=1000).strip()
            except PlaywrightTimeoutError:
                continue  # Tab became stale, skip it
            if name not in except_tabs:
                return tab
        return None

    def close_all_tabs(self, except_tabs: list[str] | None = None) -> None:
        """Close all tabs except specified ones.

        Args:
            except_tabs: Tab names to keep open. Defaults to ["Get Started"].

        Raises:
            RuntimeError: If tabs remain open after max iterations.
        """
        if except_tabs is None:
            except_tabs = ["Get Started"]

        self.layout.close_nav_drawer()

        tabs_to_close = [
            tab
            for tab in self.page.locator(".pluto-tabs-selector__btn").all()
            if tab.inner_text(timeout=1000).strip() not in except_tabs
        ]

        for _ in range(len(tabs_to_close)):
            tab = self._find_tab_to_close(except_tabs)
            if tab is None:
                return
            tab.get_by_label("pluto-tabs__close").click()
            self._dismiss_unsaved_changes_dialog()
