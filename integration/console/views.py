#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""View strip helper for Console UI automation."""

from playwright.sync_api import Locator, Page

from console.context_menu import ContextMenu
from console.layout import LayoutClient

STRIP_SELECTOR = ".console-view__views"
ITEM_SELECTOR = ".console-view__view-item"
TOOLBAR_SELECTOR = ".console-view__toolbar"
FILTER_MENU_SELECTOR = ".console-view-filter-menu"


class ViewsClient:
    """Views client for the saved-search strip at the top of an explorer.

    Views are saved searches scoped to one resource type. The strip holds a static
    "All <Resources>" view plus every saved view. Search, filters, and the create
    button are hidden until edit mode is enabled.
    """

    layout: LayoutClient
    page: Page
    ctx_menu: ContextMenu
    search_placeholder: str
    static_view_name: str

    def __init__(
        self,
        layout: LayoutClient,
        search_placeholder: str,
        static_view_name: str,
    ):
        """Initialize the views client.

        :param layout: The layout client for the console under test.
        :param search_placeholder: Placeholder of the explorer search input, used to
            detect edit mode (e.g. "Search ranges...").
        :param static_view_name: Name of the built-in view the explorer opens on
            (e.g. "All Ranges").
        """
        self.layout = layout
        self.page = layout.page
        self.ctx_menu = ContextMenu(layout.page)
        self.search_placeholder = search_placeholder
        self.static_view_name = static_view_name

    # ── Private Helpers ───────────────────────────────────────────────────

    def _strip(self) -> Locator:
        """Return the view strip locator."""
        return self.page.locator(STRIP_SELECTOR).first

    def _search_input(self) -> Locator:
        """Return the explorer search input locator."""
        return self.page.locator(f"input[placeholder='{self.search_placeholder}']")

    def _item_label(self, name: str) -> Locator:
        """Return the editable name element inside a view item."""
        return self._strip().locator(ITEM_SELECTOR).filter(has_text=name).first

    def _edit_button(self) -> Locator:
        """Return the edit mode toggle button."""
        return (
            self.page.locator("button")
            .filter(has=self.page.locator("svg.pluto-icon--edit"))
            .first
        )

    def _filter_button(self) -> Locator:
        """Return the filter dropdown trigger button."""
        return (
            self.page.locator("button")
            .filter(has=self.page.locator("svg.pluto-icon--filter"))
            .first
        )

    # ── Edit Mode ─────────────────────────────────────────────────────────

    def is_editable(self) -> bool:
        """Check whether edit mode is on.

        :returns: True if the search and filter controls are showing.
        """
        return self._search_input().is_visible()

    def enable_editing(self) -> None:
        """Turn on edit mode to show the search, filter, and create controls."""
        if self.is_editable():
            return
        self._edit_button().click()
        self._search_input().wait_for(state="visible", timeout=5000)

    # ── Search ────────────────────────────────────────────────────────────

    def wait_for_static_view(self) -> None:
        """Wait for the built-in view to show, marking the explorer as loaded."""
        self.page.get_by_text(self.static_view_name).wait_for(
            state="visible", timeout=5000
        )

    def search(self, term: str) -> None:
        """Type a search term into the explorer search input.

        :param term: The search string to type.
        """
        self.enable_editing()
        search_input = self._search_input()
        search_input.fill(term)
        search_input.dispatch_event(
            "input",
            {"bubbles": True, "data": term, "inputType": "insertText"},
        )

    def clear_search(self) -> None:
        """Clear the explorer search input."""
        self.search("")

    def open_filter(self) -> Locator:
        """Open the filter dropdown in the explorer.

        :returns: Locator for the visible filter dialog.
        """
        self.enable_editing()
        self._filter_button().click()
        dialog = self.page.locator(".pluto-dialog__dialog.pluto--visible")
        dialog.wait_for(state="visible", timeout=5000)
        return dialog

    def select_filter(self, trigger: str, option: str) -> None:
        """Select an option in one of the explorer's filter dropdowns.

        The filter is a two-level dialog: the filter button opens the first dialog,
        the trigger opens the second dialog holding the option list.

        :param trigger: Text of the dropdown trigger, e.g. "Select labels".
        :param option: Display name of the option to select.
        """
        filter_dialog = self.open_filter()
        filter_dialog.get_by_text(trigger).click()
        option_dialog = self.page.locator(".pluto-select__dialog.pluto--visible")
        option_dialog.wait_for(state="visible", timeout=5000)
        item = option_dialog.locator(".pluto-list__item").filter(has_text=option).first
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.layout.press_escape()
        self.layout.press_escape()

    def clear_filter(self, option: str) -> None:
        """Remove an option from the active filter by clicking its tag close button.

        :param option: Display name of the option to remove.
        """
        tag = self.page.locator(".pluto-tag:has(button)").filter(has_text=option).first
        tag.wait_for(state="visible", timeout=5000)
        tag.hover()
        tag.locator("button").click()
        tag.wait_for(state="hidden", timeout=5000)
