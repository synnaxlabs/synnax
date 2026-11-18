#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal

from playwright.sync_api import Locator

from console.range.surface import Surface

Group = Literal["Running", "Favorites", "Recent"]
"""A group of the ranges toolbar. A range can show in more than one group."""


class Toolbar(Surface):
    """The ranges toolbar in the left sidebar. It lists the running, favorite, and
    recent ranges in three groups."""

    ROOT_SELECTOR = ".console-range-toolbar"
    DRAFT_INPUT_SELECTOR = ".console-range-toolbar__draft-input input"

    def show(self) -> None:
        """Show the ranges toolbar in the left sidebar."""
        self.layout.show_resource_toolbar("Ranges")

    def hide(self) -> None:
        """Hide the ranges toolbar."""
        self.layout.close_left_toolbar()

    def _group(self, group: Group) -> Locator:
        return self.layout.page.locator(
            f"{self.ROOT_SELECTOR} section[aria-label='{group}']"
        )

    def expand(self, group: Group) -> None:
        """Open a group of the toolbar. A collapsed group stays collapsed across
        sessions, so a test cannot count on it being open.

        :param group: The group to open.
        """
        heading = self._group(group).locator(
            ".console-range-toolbar__group-header [aria-expanded='false']"
        )
        if heading.count() > 0:
            heading.click(timeout=5000)

    def get_item(self, name: str, *, group: Group | None = None) -> Locator:
        """Get the line of a range in the toolbar.

        :param name: The exact name of the range.
        :param group: The group to look in. None looks in every group.
        """
        scope = (
            self.layout.page.locator(self.ROOT_SELECTOR)
            if group is None
            else self._group(group)
        )
        return scope.get_by_role("listitem", name=name, exact=True).first

    def get_item_time(self, name: str, *, group: Group = "Favorites") -> str:
        """Get the time text of a range line: elapsed time in Running, the start in
        Favorites, and the time since the end in Recent.

        :param name: The name of the range.
        :param group: The group to read the line from.
        """
        self.show()
        self.expand(group)
        item = self.get_item(name, group=group)
        return item.locator(".console-range-toolbar__meta").inner_text()

    def exists(self, name: str, *, group: Group | None = None) -> bool:
        """Check if a range shows in the toolbar.

        :param name: The name of the range.
        :param group: The group to look in. None looks in every group.
        """
        self.show()
        if group is not None:
            self.expand(group)
        return self.layout.locator_exists(self.get_item(name, group=group))

    def wait_for_removed(self, name: str, *, group: Group | None = None) -> None:
        """Wait for a range to leave the toolbar.

        :param name: The name of the range.
        :param group: The group to watch. None waits until no group shows it.
        """
        self.show()
        self.layout.wait_for_hidden(self.get_item(name, group=group))

    def unfavorite(self, name: str) -> None:
        """Remove a range from the favorites with the star on its line."""
        self.show()
        self.expand("Favorites")
        item = self.get_item(name, group="Favorites")
        item.wait_for(state="visible", timeout=5000)
        # The star's icon paints over its input, so the event goes to the input.
        item.get_by_role("checkbox", name="Favorite").dispatch_event("click")
        self.wait_for_removed(name, group="Favorites")

    def plot(self, name: str) -> None:
        """Open a new line plot of a range from the context menu of its line."""
        self.show()
        self.expand("Favorites")
        item = self.get_item(name, group="Favorites")
        item.wait_for(state="visible", timeout=5000)
        self.ctx_menu.action(item, "Plot")

    def start(self, name: str) -> None:
        """Start a range now with the Start button in the header.

        :param name: The name of the new range.
        """
        self.show()
        self.layout.page.get_by_role("button", name="Start range", exact=True).click(
            timeout=5000
        )
        self._save_draft(name)

    def start_child(self, parent: str, name: str) -> None:
        """Start a child range now under a running range, with the button on the
        parent's line.

        :param parent: The name of the running parent range.
        :param name: The name of the new child range.
        """
        self.show()
        self.expand("Running")
        item = self.get_item(parent, group="Running")
        item.hover()
        item.get_by_role("button", name="Start child range", exact=True).click(
            timeout=5000
        )
        self._save_draft(name)

    def next(self, name: str, next_name: str) -> str:
        """Stop a running child range and start the range after it, with the button
        on its line.

        :param name: The name of the running child range.
        :param next_name: The name of the range that starts after it.
        :returns: The name that the draft suggested before the fill.
        """
        self.show()
        self.expand("Running")
        item = self.get_item(name, group="Running")
        item.hover()
        item.get_by_role("button", name="Stop and start next", exact=True).click(
            timeout=5000
        )
        return self._save_draft(next_name)

    def stop(self, name: str) -> None:
        """Stop a running range with the button on its line."""
        self.show()
        self.expand("Running")
        item = self.get_item(name, group="Running")
        item.hover()
        item.get_by_role("button", name="Stop", exact=True).click(timeout=5000)
        self.wait_for_removed(name, group="Running")

    def _save_draft(self, name: str) -> str:
        """Name the open draft and save it with Enter.

        :returns: The name that the draft suggested. The draft opens with it set.
        """
        draft = self.layout.page.locator(self.DRAFT_INPUT_SELECTOR)
        draft.wait_for(state="visible", timeout=5000)
        suggested = draft.input_value()
        draft.fill(name)
        draft.press("Enter")
        draft.wait_for(state="detached", timeout=5000)
        return suggested
