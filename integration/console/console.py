#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Generator
from contextlib import contextmanager

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
from .workspace import PageType, WorkspaceClient


class Console:
    """
    Console UI automation interface.

    Provides utility methods for interacting with the Synnax Console application
    via Playwright, including page management, keyboard shortcuts, form interactions,
    and element clicking helpers.
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
        self.notifications = NotificationsClient(self.layout)
        self.docs = DocsClient(self.layout)
        self.labels = LabelClient(self.layout)
        self.rack = RackClient(self.layout)
        self.arc = ArcClient(self.layout, self.notifications)
        self.access = AccessClient(self.layout, self.notifications)
        self.channels = ChannelClient(self.layout, self.notifications, self)
        self.ranges = RangesClient(self.layout, self.notifications, self)
        self.workspace = WorkspaceClient(self.layout, self)

    def command_palette(self, command: str, retries: int = 3) -> None:
        """Execute a command via the command palette."""
        self.layout.command_palette(command, retries)

    def search_palette(self, query: str, retries: int = 3) -> None:
        """Search for a resource via the command palette (without > prefix)."""
        self.layout.search_palette(query, retries)

    @property
    def ESCAPE(self) -> None:
        self.layout.press_escape()

    @property
    def ENTER(self) -> None:
        self.layout.press_enter()

    @property
    def META_ENTER(self) -> None:
        self.layout.press_meta_enter()

    @property
    def DELETE(self) -> None:
        self.layout.press_delete()

    def select_all(self) -> None:
        """Select all text in the focused element."""
        self.layout.select_all()

    def select_all_and_type(self, text: str) -> None:
        """Select all text in the focused element and type new text."""
        self.layout.select_all_and_type(text)

    @property
    def MODAL_OPEN(self) -> bool:
        return self.layout.is_modal_open()

    def show_resource_toolbar(self, resource: str) -> None:
        """Show a resource toolbar by clicking its icon in the sidebar."""
        self.layout.show_resource_toolbar(resource)

    def close_nav_drawer(self) -> None:
        """Close any open side nav drawer (left/right, not bottom visualization toolbar)."""
        self.layout.close_nav_drawer()

    def select_from_dropdown(self, text: str, placeholder: str | None = None) -> None:
        """Select an item from an open dropdown."""
        self.layout.select_from_dropdown(text, placeholder)

    def create_page(
        self, page_type: PageType, page_name: str | None = None
    ) -> tuple[Locator, str]:
        """Create a new page via New Page (+) button or command palette (randomly chosen)."""
        return self.workspace.create_page(page_type, page_name)

    def close_page(self, page_name: str) -> None:
        """Close a page by name. Ignores unsaved changes."""
        self.workspace.close_page(page_name)

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

    def click_btn(self, button_label: str) -> None:
        """Click a button by label."""
        self.layout.click_btn(button_label)

    def get_toggle(self, toggle_label: str) -> bool:
        """Get the value of a toggle by label."""
        return self.layout.get_toggle(toggle_label)

    def click_checkbox(self, checkbox_label: str) -> None:
        """Click a checkbox by label."""
        self.layout.click_checkbox(checkbox_label)

    def fill_input_field(self, input_label: str, value: str) -> None:
        """Fill an input field by label."""
        self.layout.fill_input_field(input_label, value)

    def get_input_field(self, input_label: str) -> str:
        """Get the value of an input field by label."""
        return self.layout.get_input_field(input_label)

    def get_dropdown_value(self, dropdown_label: str) -> str:
        """Get the current value of a dropdown by label."""
        return self.layout.get_dropdown_value(dropdown_label)

    def get_selected_button(self, button_options: list[str]) -> str:
        """Get the currently selected button from a button group (no label)."""
        return self.layout.get_selected_button(button_options)

    def click(
        self, selector: str | Locator, timeout: int = 500, sleep: int = 100
    ) -> None:
        """Click an element by text selector or Locator."""
        self.layout.click(selector, timeout, sleep)

    def meta_click(
        self, selector: str | Locator, timeout: int = 500, sleep: int = 100
    ) -> None:
        """Click an element with platform-appropriate modifier key held."""
        self.layout.meta_click(selector, timeout, sleep)

    def check_for_modal(self) -> bool:
        """Check for a modal."""
        return self.layout.check_for_modal()

    @contextmanager
    def bring_to_front(self, element: Locator) -> Generator[Locator, None, None]:
        """Context manager that temporarily brings an element to the front."""
        with self.layout.bring_to_front(element) as el:
            yield el

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

        self.close_nav_drawer()

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
