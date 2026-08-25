#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""View strip helper for Console UI automation."""

import re

from playwright.sync_api import Locator, expect

from console.base import ResourceClient
from console.layout import LayoutClient

STRIP_SELECTOR = ".console-view__views"
ITEM_SELECTOR = ".pluto-tabs__tab"
# Every list item names its text element after its key, per List.itemNameID.
ITEM_LABEL_SELECTOR = "[id$='-name']"
SELECTED_CLASS = "pluto--selected"


class ViewsClient(ResourceClient):
    """Views client for the saved-search strip at the top of an explorer.

    Views are saved searches scoped to one resource type. The strip holds a static
    "All <resources>" view plus every saved view. Search, filters, and the create
    button are hidden until edit mode is enabled.
    """

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
            (e.g. "All ranges").
        """
        super().__init__(layout)
        self.search_placeholder = search_placeholder
        self.static_view_name = static_view_name

    # ── Private Helpers ───────────────────────────────────────────────────

    def _strip(self) -> Locator:
        """Return the view strip locator."""
        return self.page.locator(STRIP_SELECTOR).first

    def _search_input(self) -> Locator:
        """Return the explorer search input locator."""
        return self.page.get_by_placeholder(self.search_placeholder)

    def _item_label(self, name: str) -> Locator:
        """Return an exact-name label filter, relative to whatever it is nested in."""
        pattern = re.compile(rf"^{re.escape(name)}$")
        return self.page.locator(ITEM_LABEL_SELECTOR).filter(has_text=pattern)

    # ── View Items ────────────────────────────────────────────────────────

    def get_view_item(self, name: str) -> Locator:
        """Return the clickable view item with the given name.

        :param name: Exact name of the view.
        """
        return self._strip().locator(ITEM_SELECTOR).filter(has=self._item_label(name))

    def get_view_items(self) -> list[str]:
        """Return the names of every view in the strip, in display order."""
        return self._strip().locator(ITEM_LABEL_SELECTOR).all_inner_texts()

    def wait_for_static_view(self) -> None:
        """Wait for the built-in view to show, marking the explorer as loaded."""
        self.page.get_by_text(self.static_view_name).wait_for(
            state="visible", timeout=5000
        )

    def exists(self, name: str) -> bool:
        """Check whether a view with the given name is in the strip."""
        return self.layout.locator_exists(self.get_view_item(name))

    def wait_for(self, name: str) -> None:
        """Wait for a view to appear in the strip."""
        self.get_view_item(name).wait_for(state="visible", timeout=5000)

    def wait_for_removed(self, name: str) -> None:
        """Wait for a view to leave the strip."""
        self.layout.wait_for_hidden(self.get_view_item(name))

    # ── Select ────────────────────────────────────────────────────────────

    def select(self, name: str) -> None:
        """Switch to a view by clicking its item.

        :param name: Exact name of the view.
        """
        item = self.get_view_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()
        expect(item).to_have_class(re.compile(SELECTED_CLASS), timeout=5000)

    def get_selected(self) -> str:
        """Return the name of the currently selected view."""
        selected = self._strip().locator(f"{ITEM_SELECTOR}.{SELECTED_CLASS}").first
        selected.wait_for(state="visible", timeout=5000)
        return selected.locator(ITEM_LABEL_SELECTOR).inner_text()

    def is_selected(self, name: str) -> bool:
        """Check whether the named view is the selected one."""
        return self.get_selected() == name

    # ── Create / Rename / Delete ──────────────────────────────────────────

    def create(self, name: str) -> None:
        """Create a view from the current search and filters.

        The create button pops a rename modal prefilled with a default name.

        :param name: Name to give the new view.
        """
        self.enable_editing()
        self.page.get_by_role("button", name="Create view", exact=True).first.click()
        modal = self.page.locator(LayoutClient.MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        modal.locator("input[placeholder='Name']").fill(name)
        modal.get_by_role("button", name="Save", exact=True).click(timeout=5000)
        modal.wait_for(state="hidden", timeout=5000)
        self.wait_for(name)

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename a view via its context menu. The name edits in place.

        :param old_name: Current name of the view.
        :param new_name: Name to set.
        """
        self.layout.rename_in_place(self.get_view_item(old_name), new_name)
        self.wait_for(new_name)

    def delete(self, name: str) -> None:
        """Delete a view via its context menu and confirm.

        :param name: Exact name of the view.
        """
        item = self.get_view_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.layout.delete_with_confirmation(item)
        self.wait_for_removed(name)

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
        self.page.get_by_role("button", name="Enable editing", exact=True).first.click()
        self._search_input().wait_for(state="visible", timeout=5000)

    def disable_editing(self) -> None:
        """Turn off edit mode, hiding the search, filter, and create controls."""
        if not self.is_editable():
            return
        self.page.get_by_role(
            "button", name="Disable editing", exact=True
        ).first.click()
        self._search_input().wait_for(state="hidden", timeout=5000)

    # ── Search ────────────────────────────────────────────────────────────

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

    # ── Filters ───────────────────────────────────────────────────────────

    def open_filter(self) -> Locator:
        """Open the filter dropdown in the explorer.

        :returns: Locator for the visible filter dialog.
        """
        self.enable_editing()
        self.page.get_by_role("button", name="Filter", exact=True).first.click()
        dialog = self.layout.dialog
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
        item = self.layout.dialog.get_by_role("option").filter(has_text=option).first
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.layout.press_escape()
        self.layout.press_escape()

    def clear_filter(self, option: str) -> None:
        """Remove an option from the active filter by clicking its chip close button.

        :param option: Display name of the option to remove.
        """
        remove_btn = self.page.get_by_role(
            "button", name=f"Remove {option}", exact=True
        ).first
        remove_btn.wait_for(state="visible", timeout=5000)
        remove_btn.dispatch_event("click")
        remove_btn.wait_for(state="hidden", timeout=5000)
