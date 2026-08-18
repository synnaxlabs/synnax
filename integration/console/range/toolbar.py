#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import Locator

from console.range.surface import UNFAVORITE_ACTIONS, Surface
from x.color import Color


class Toolbar(Surface):
    """The ranges toolbar in the left sidebar. Shows only favorited ranges."""

    ITEM_SELECTOR = ".console-range-list-item"

    def show(self) -> None:
        """Show the ranges toolbar in the left sidebar (favorites only)."""
        self.layout.show_resource_toolbar("Ranges")

    def hide(self) -> None:
        """Hide the ranges toolbar."""
        self.layout.close_left_toolbar()

    def get_item(self, name: str) -> Locator:
        """Get a range item locator from the toolbar by name."""
        return self.layout.get_list_item(self.ITEM_SELECTOR, name)

    def get_item_time(self, name: str) -> str:
        """Get the displayed time text from a toolbar range item.

        :param name: The name of the range.
        :returns: The time range text (e.g. "Today 16:23:35 → 16:23:35").
        """
        self.show()
        item = self.get_item(name)
        return item.locator("small.pluto-text--small").first.inner_text()

    def exists(self, name: str) -> bool:
        """Check if a range exists in the toolbar (is favorited)."""
        self.show()
        return self.layout.locator_exists(self.get_item(name))

    def wait_for_removed(self, name: str) -> None:
        """Wait for a range to be removed from the toolbar."""
        self.show()
        self.layout.wait_for_hidden(self.get_item(name))

    def unfavorite(self, name: str) -> None:
        """Remove a range from favorites via context menu in the toolbar."""
        self.show()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        self._ctx_action_any(item, UNFAVORITE_ACTIONS)
        self.wait_for_removed(name)

    def _ctx_action(self, name: str, action_text: str) -> None:
        """Show the toolbar, find an item by name, and run a context menu
        action."""
        self.show()
        item = self.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, action_text)

    def save_to_synnax(self, name: str) -> None:
        """Save a local range to Synnax via context menu in the toolbar."""
        self._ctx_action(name, "Save to Core")

    def add_to_new_plot(self, name: str) -> None:
        """Add a range to a new line plot via context menu in the toolbar."""
        self._ctx_action(name, "Add to new plot")

    def add_to_active_plot(self, name: str) -> None:
        """Add a range to the active line plot via context menu in the
        toolbar."""
        self._ctx_action(name, "Add to active plot")

    def get_label(self, range_name: str, label_name: str) -> Locator:
        """Get a label tag within a range item in the toolbar."""
        range_item = self.get_item(range_name)
        return range_item.locator(".pluto-tag").filter(has_text=label_name).first

    def label_exists(self, range_name: str, label_name: str) -> bool:
        """Check if a label exists on a range in the toolbar."""
        self.show()
        return self.layout.locator_exists(self.get_label(range_name, label_name))

    def wait_for_label_removed(self, range_name: str, label_name: str) -> None:
        """Wait until a label is removed from a range in the toolbar."""
        self.show()
        self.layout.wait_for_hidden(self.get_label(range_name, label_name))

    def get_label_color(self, range_name: str, label_name: str) -> Color | None:
        """Get the color of a label's icon in the range toolbar."""
        self.show()
        label = self.get_label(range_name, label_name)
        if label.count() == 0:
            return None
        icon = label.locator("svg").first
        if icon.count() == 0:
            return None
        color = icon.get_attribute("color")
        if color is None:
            return None
        return Color(color)

    def get_all_labels(self, range_name: str) -> list[str]:
        """Get all labels currently visible for a range in the toolbar.

        :param range_name: The name of the range to check.
        :returns: List of label names currently displayed for this range.
        """
        self.show()
        range_item = self.get_item(range_name)
        if not self.layout.locator_exists(range_item):
            return []

        label_tags = range_item.locator(".pluto-tag")
        label_count = label_tags.count()

        labels = []
        for i in range(label_count):
            label_text = label_tags.nth(i).text_content()
            if label_text:
                labels.append(label_text.strip())

        return labels
