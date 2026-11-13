#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import time
from re import search as re_search

from playwright.sync_api import Locator, TimeoutError as PlaywrightTimeoutError, expect

import synnax as sy
from console.base import ResourceClient
from console.color import pick as pick_color
from console.layout import LayoutClient
from x.color import Color

_ITEM_SELECTOR = ".console-label-settings__item"
_SEARCH_PLACEHOLDER = "Search labels"
_POLL_MS = 100


class LabelClient(ResourceClient):
    """Console label client for managing labels on the Labels page of Settings."""

    def __init__(self, *, layout: LayoutClient, client: sy.Synnax) -> None:
        super().__init__(layout)
        self.client = client

    def create(self, name: str, *, color: str | None = None) -> None:
        """Create a new label.

        :param name: The name for the new label.
        :param color: Optional hex color code (e.g., "#FF0000") to set for the new
            label.
        """
        self._show_page()
        self.page.get_by_role("button", name="Create label", exact=True).click()
        modal = self.page.locator(self.layout.MODAL_SELECTOR).first
        name_input = modal.get_by_placeholder("Label name", exact=True)
        name_input.wait_for(state="visible", timeout=5000)
        if color is not None:
            pick_color(self.page, modal.get_by_label("Label color"), hex_color=color)
        name_input.fill(name)
        modal.get_by_role("button", name="Create").click()
        modal.wait_for(state="hidden", timeout=5000)
        self._get_item(name)

    def exists(self, name: str) -> bool:
        """Check if a label exists by name."""
        return self._find_item(name) is not None

    def get_color(self, name: str) -> Color:
        """Get the color of a label by name."""
        swatch = self._get_item(name).get_by_label("Color", exact=True)
        style = swatch.get_attribute("style")
        if style is None:
            raise ValueError(f"Label '{name}' has no style attribute")
        match = re_search(r"--pluto-swatch-color:\s*(rgba?\([^)]+\))", style)
        if match is None:
            raise ValueError(f"Label '{name}' does not have --pluto-swatch-color")
        return Color(match.group(1))

    def rename(self, *, old_name: str, new_name: str) -> None:
        """Rename an existing label.

        :param old_name: The current name of the label.
        :param new_name: The new name for the label.
        """
        self.layout.rename_in_place(self._get_item(old_name), new_name)
        # The Console holds an edit for a short window before it sends it, so a search
        # for the new name before the Core has it comes back empty.
        deadline = time.monotonic() + 5
        while not self.client.labels.retrieve(names=[new_name]):
            if time.monotonic() >= deadline:
                raise PlaywrightTimeoutError(
                    f"The Core did not rename label to {new_name}"
                )
            self.page.wait_for_timeout(_POLL_MS)
        self._get_item(new_name)

    def delete(self, name: str) -> None:
        """Delete a label.

        :param name: The name of the label to delete.
        :raises ValueError: If the label with the provided name is not found.
        """
        self.layout.delete_with_confirmation(self._get_item(name))
        expect(self._items(name)).to_have_count(0, timeout=5000)

    def list_all(self) -> list[str]:
        """List all existing labels.

        :returns: List of label names.
        """
        self._search("")
        items = self.page.locator(_ITEM_SELECTOR)
        try:
            items.first.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError:
            return []
        return [
            name for item in items.all() if (name := item.get_attribute("aria-label"))
        ]

    def change_color(self, *, name: str, new_color: str) -> None:
        """Change the color of a label.

        :param name: The name of the label to update.
        :param new_color: The new hex color code (e.g., "#FF0000").
        """
        swatch = self._get_item(name).get_by_label("Color", exact=True)
        pick_color(self.page, swatch, hex_color=new_color)

    def _show_page(self) -> None:
        """Show the Labels page of Settings."""
        nav = self.layout.open_settings()
        nav.get_by_role("menuitem", name="Labels", exact=True).click()
        self.page.get_by_placeholder(_SEARCH_PLACEHOLDER, exact=True).wait_for(
            state="visible", timeout=5000
        )

    def _search(self, term: str) -> None:
        self._show_page()
        self.page.get_by_placeholder(_SEARCH_PLACEHOLDER, exact=True).fill(term)

    def _items(self, name: str) -> Locator:
        """The lines on the Labels page whose name is ``name``."""
        return self.page.locator(_ITEM_SELECTOR).and_(
            self.page.get_by_role("option", name=name, exact=True)
        )

    def _find_item(self, name: str) -> Locator | None:
        """Search the Labels page for a label, or return None if it is not listed."""
        self._search(name)
        item = self._items(name).first
        try:
            item.wait_for(state="visible", timeout=5000)
        except PlaywrightTimeoutError:
            return None
        return item

    def _get_item(self, name: str) -> Locator:
        """Like _find_item, but raises if the search does not list the label."""
        item = self._find_item(name)
        if item is None:
            raise ValueError(f"Label '{name}' not found on the Labels page")
        return item
