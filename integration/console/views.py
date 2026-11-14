#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""View strip helper for Console UI automation."""

from playwright.sync_api import Locator, expect

from console.base import ResourceClient
from console.layout import LayoutClient


class ViewsClient(ResourceClient):
    """Views client for the saved-search strip at the top of an explorer.

    Views are saved searches scoped to one resource type. The strip holds a static "All
    <resources>" view plus every saved view, after the search and before the create
    button. The filters sit in the bar under the strip.
    """

    search_placeholder: str
    static_view_name: str

    def __init__(
        self,
        *,
        layout: LayoutClient,
        search_placeholder: str,
        static_view_name: str,
    ):
        """Initialize the views client.

        :param layout: The layout client for the console under test.
        :param search_placeholder: Placeholder of the explorer search input (e.g.
            "Search ranges...").
        :param static_view_name: Name of the built-in view the explorer opens on
            (e.g. "All ranges").
        """
        super().__init__(layout)
        self.search_placeholder = search_placeholder
        self.static_view_name = static_view_name

    # ── Private Helpers ───────────────────────────────────────────────────

    def _strip(self) -> Locator:
        """Return the view strip locator.

        The name separates the strip from the panel tab strip, which is the other
        tablist on the page.
        """
        return self.page.get_by_role("tablist", name="Views").first

    def _search_input(self) -> Locator:
        """Return the explorer search input locator."""
        return self.page.get_by_placeholder(self.search_placeholder)

    # ── View Items ────────────────────────────────────────────────────────

    def get_view_item(self, name: str) -> Locator:
        """Return the clickable view item with the given name.

        :param name: Exact name of the view.
        """
        return self._strip().get_by_role("tab", name=name, exact=True)

    def get_view_items(self) -> list[str]:
        """Return the names of every view in the strip, in display order."""
        return self._strip().get_by_role("tab").all_inner_texts()

    def wait_for_static_view(self) -> None:
        """Wait for the built-in view to show, marking the explorer as loaded."""
        self.get_view_item(self.static_view_name).wait_for(
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
        expect(item).to_have_attribute("aria-selected", "true", timeout=5000)

    def get_selected(self) -> str:
        """Return the name of the currently selected view."""
        selected = self._strip().get_by_role("tab", selected=True).first
        selected.wait_for(state="visible", timeout=5000)
        return selected.inner_text()

    def is_selected(self, name: str) -> bool:
        """Check whether the named view is the selected one."""
        return self.get_selected() == name

    # ── Create / Rename / Delete ──────────────────────────────────────────

    def create(self, name: str) -> None:
        """Create a view from the current search and filters.

        The create button opens a modal that asks for the name of the view.

        :param name: Name to give the new view.
        """
        self.page.get_by_role("button", name="Create view", exact=True).first.click()
        modal = self.layout.dialog
        modal.wait_for(state="visible", timeout=5000)
        modal.get_by_placeholder("View name", exact=True).fill(name)
        modal.get_by_role("button", name="Create", exact=True).click(timeout=5000)
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

    # ── Search ────────────────────────────────────────────────────────────

    def search(self, term: str) -> None:
        """Type a search term into the explorer search input.

        :param term: The search string to type.
        """
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
        self.page.get_by_role("button", name="Filter", exact=True).first.click()
        dialog = self.layout.dialog
        dialog.wait_for(state="visible", timeout=5000)
        return dialog

    def select_filter(self, trigger: str, option: str) -> None:
        """Select an option in one of the explorer's filter dropdowns.

        The filter button opens the filter menu, and the menu item of the property
        opens its picker beside the menu.

        :param trigger: Name of the property's menu item, e.g. "Labels".
        :param option: Display name of the option to select.
        """
        filter_dialog = self.open_filter()
        filter_dialog.get_by_role("menuitem", name=trigger, exact=True).click()
        item = self.layout.dialog.get_by_role("option").filter(has_text=option).first
        item.wait_for(state="visible", timeout=5000)
        item.click()
        self.layout.press_escape()
        self.layout.press_escape()

    def clear_filter(self, prop: str) -> None:
        """Remove a property from the active filter with the remove button of its chip.

        :param prop: Name of the property the chip shows, e.g. "Labels".
        """
        remove_btn = self.page.get_by_role(
            "button", name=f"Remove {prop} filter", exact=True
        ).first
        remove_btn.wait_for(state="visible", timeout=5000)
        remove_btn.dispatch_event("click")
        remove_btn.wait_for(state="hidden", timeout=5000)
