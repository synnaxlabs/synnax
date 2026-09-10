#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator, expect

import synnax as sy
from console.layout import LayoutClient
from console.range.surface import FAVORITE_ACTIONS, UNFAVORITE_ACTIONS, Surface
from console.range.toolbar import Toolbar
from console.views import ViewsClient


class Explorer(Surface):
    """The Range Explorer page. Shows all persisted ranges."""

    SEARCH_INPUT_PLACEHOLDER = "Search ranges..."
    STATIC_VIEW_NAME = "All ranges"
    # The explorer list must stay the same height at the bottom for longer than a page
    # fetch before it counts as fully paged in.
    SCROLL_SETTLE = 1500 * sy.TimeSpan.MILLISECOND

    def __init__(self, layout: LayoutClient, toolbar: Toolbar):
        super().__init__(layout)
        self.toolbar = toolbar
        self.views = ViewsClient(
            layout, self.SEARCH_INPUT_PLACEHOLDER, self.STATIC_VIEW_NAME
        )

    def open(self) -> None:
        """Open the Range Explorer page (shows all ranges)."""
        self.layout.command_palette("Open range explorer")
        self.views.wait_for_static_view()

    def _list(self) -> Locator:
        """Get the explorer's scrolling list of ranges.

        The name separates the list from the other range lists on the page, such as the
        toolbar's favorites.
        """
        return self.layout.page.get_by_role("list", name="Ranges").first

    def get_mounted_item(self, name: str) -> Locator:
        """Get a range row already mounted in the list, skipping the scroll sweep.

        :param name: Exact name of the range.
        """
        return self._list().get_by_role("listitem", name=name, exact=True).first

    def get_item(self, name: str) -> Locator:
        """Get a range item locator from the explorer by name."""
        item = self.get_mounted_item(name)
        if item.count() == 0:
            self._scroll_to(item)
        return item

    def _scroll_to(
        self, item: Locator, budget: sy.TimeSpan = 30 * sy.TimeSpan.SECOND
    ) -> None:
        """Scroll the explorer list until item renders, or the list runs out.

        The explorer mounts only the rows in view and pages in more as it nears
        the bottom, so a range further down has no element to wait on. Filtering
        the list by name would find it too, but would hide every other range,
        which breaks selecting several at once.

        :param item: Locator for the range row to reveal.
        :param budget: Backstop on the whole sweep. Callers that probe for a
            range that is not there page through the entire list.
        """
        lst = self._list()
        if lst.count() == 0:
            return
        lst.evaluate("el => { el.scrollTop = 0; }")
        timer = sy.Timer()
        settle = sy.Timer()
        prev_height = -1
        while timer.elapsed() < budget:
            if item.count() > 0:
                return
            height = lst.evaluate("el => el.scrollHeight")
            at_bottom = lst.evaluate(
                "el => el.scrollTop + el.clientHeight >= el.scrollHeight - 1"
            )
            # The pager appends on reaching the bottom, so only give up once the list
            # stops growing there. A short pause proves nothing: the next page can
            # still be in flight.
            if not at_bottom or height != prev_height:
                settle.reset()
            elif settle.elapsed() >= self.SCROLL_SETTLE:
                return
            prev_height = height
            lst.evaluate("el => { el.scrollTop += el.clientHeight * 0.8; }")
            self.layout.page.wait_for_timeout(150)

    def get_item_time(self, name: str) -> str:
        """Get the displayed time text from an explorer range item.

        :param name: The name of the range.
        :returns: The time range text (e.g. "Jan 1 00:00:00 → Jan 2 00:00:00").
        """
        item = self.get_item(name)
        return item.get_by_role("group", name="Time range").inner_text()

    def exists(self, name: str) -> bool:
        """Check if a range exists in the explorer."""
        return self.layout.locator_exists(self.get_item(name))

    def wait_for_removed(self, name: str) -> None:
        """Wait for a range to be removed from the explorer."""
        self.layout.wait_for_hidden(self.get_item(name))

    def rename(self, old_name: str, new_name: str) -> None:
        """Rename a range from the explorer. The name edits in place."""
        self.layout.rename_in_place(self.get_item(old_name), new_name)

    def delete(self, name: str) -> None:
        """Delete a range via context menu in the explorer."""
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.layout.delete_with_confirmation(item)
        item.wait_for(state="hidden", timeout=5000)

    def favorite(self, name: str) -> None:
        """Add a range to favorites via context menu in the explorer."""
        self.layout.hide_visualization_toolbar()
        self.notifications.close_all()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.open_on(item)
        menu = self.layout.page.get_by_role("menu")
        add_btn = self._any_text_locator(menu, FAVORITE_ACTIONS)
        remove_btn = self._any_text_locator(menu, UNFAVORITE_ACTIONS)
        add_btn.or_(remove_btn).wait_for(state="visible", timeout=2000)
        if remove_btn.is_visible():
            self.ctx_menu.close()
            return
        self._click_visible_option(FAVORITE_ACTIONS)

    def unfavorite(self, name: str) -> None:
        """Remove a range from favorites via context menu in the explorer."""
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        self._ctx_action_any(item, UNFAVORITE_ACTIONS)
        self.toolbar.wait_for_removed(name)

    def _deselect_all(self) -> None:
        """Deselect all explorer ranges by clicking their checked checkboxes.

        The checkbox indicator paints over the input, so a real click lands on the wrong
        element. The event goes straight to the input instead.
        """
        checked = self._list().get_by_role("checkbox", name="Select", checked=True)
        # Wait out each unchecking before the next. Clicking again while the row is
        # still checked re-selects the box that was just cleared.
        while (remaining := checked.count()) > 0:
            checked.first.dispatch_event("click")
            expect(checked).to_have_count(remaining - 1, timeout=5000)

    def _select_many(self, names: list[str]) -> Locator:
        """Select multiple explorer ranges via their checkbox labels."""
        return self.layout.select_items(names, self.get_item)

    def favorite_many(self, names: list[str]) -> None:
        """Favorite multiple ranges via multi-select context menu in the
        explorer.

        :param names: The names of the ranges to favorite.
        """
        if not names:
            return
        last_item = self._select_many(names)
        self._ctx_action_any(last_item, FAVORITE_ACTIONS)
        self._deselect_all()

    def unfavorite_many(self, names: list[str]) -> None:
        """Unfavorite multiple ranges via multi-select context menu in the
        explorer.

        :param names: The names of the ranges to unfavorite.
        """
        if not names:
            return
        last_item = self._select_many(names)
        self._ctx_action_any(last_item, UNFAVORITE_ACTIONS)
        self._deselect_all()
        for name in names:
            self.toolbar.wait_for_removed(name)

    def delete_many(self, names: list[str]) -> None:
        """Delete multiple ranges via multi-select context menu in the
        explorer."""
        if not names:
            return
        last_item = self._select_many(names)
        self.ctx_menu.action(last_item, "Delete")
        self.layout.confirm_delete()
        for name in names:
            self.wait_for_removed(name)

    def copy_link(self, name: str) -> None:
        """Copy link to a range via context menu in the explorer.

        :param name: The name of the range.
        """
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Copy link")

    def create_child_range(self, parent_name: str, child_name: str) -> None:
        """Create a child range via context menu in the explorer.

        :param parent_name: The name of the parent range.
        :param child_name: The name for the new child range.
        """
        item = self.get_item(parent_name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Create child range")
        self.fill_create_modal(child_name)

    def search(self, term: str) -> None:
        """Type a search term in the explorer search input.

        :param term: The search string to type.
        """
        self.views.search(term)

    def clear_search(self) -> None:
        """Clear the explorer search input."""
        self.views.clear_search()

    def select_label_filter(self, label_name: str) -> None:
        """Select a label in the explorer's label filter dropdown.

        :param label_name: The name of the label to select.
        """
        self.views.select_filter("Select labels", label_name)

    def clear_label_filter(self, label_name: str) -> None:
        """Remove a label from the active filter by clicking its chip close
        button.

        :param label_name: The name of the label chip to remove.
        """
        self.views.clear_filter(label_name)
