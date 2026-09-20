#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING

from playwright.sync_api import expect

from console.layout import LayoutClient
from console.range.explorer import Explorer
from console.range.overview import Overview
from console.range.surface import CREATE_MODAL_SELECTOR, Surface
from console.range.toolbar import Toolbar

if TYPE_CHECKING:
    from console.pages import PagesClient


class Client(Surface):
    """Console ranges client. Surface-specific operations live on
    ``toolbar``, ``explorer``, and ``overview``."""

    def __init__(
        self,
        layout: LayoutClient,
        pages: "PagesClient",
    ):
        super().__init__(layout)
        self.pages = pages
        self.toolbar = Toolbar(layout)
        self.explorer = Explorer(layout, self.toolbar)
        self.overview = Overview(layout, self.toolbar, self.explorer)

    def create(
        self,
        name: str,
        *,
        persisted: bool = True,
        parent: str | None = None,
        labels: list[str] | None = None,
        stage: str | None = None,
    ) -> None:
        """Create a new range via the command palette.

        :param name: The name for the new range.
        :param persisted: If True, saves to the Core. If False, saves locally
            only.
        :param parent: Optional parent range name to set.
        :param labels: Optional list of label names to add.
        :param stage: Optional stage to set ("To do", "In progress",
            "Completed").
        """
        self.layout.command_palette("Create range")
        modal = self.layout.page.locator(CREATE_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        self.fill_create_modal(
            name, persisted=persisted, parent=parent, labels=labels, stage=stage
        )

    def set_active(self, name: str) -> None:
        """Set a range as the active range (from toolbar)."""
        self.toolbar.show()
        item = self.toolbar.get_item(name)
        item.wait_for(state="visible", timeout=5000)
        item.click()

    def favorite(self, name: str) -> None:
        """Favorite a range from the explorer (idempotent).

        The overview no longer carries a favorite control; favoriting happens
        on list items in the explorer and toolbar.

        :param name: The name of the range to favorite.
        """
        self.explorer.open()
        self.explorer.favorite(name)
        self.layout.close_tab("Range explorer")

    def open_from_search(self, name: str) -> None:
        """Open a range overview by searching its name in the command palette.

        :param name: Name of the range to search for and open.
        """
        self.layout.search_palette(name)
        name_input = self.layout.page.locator("input[placeholder='Name']:visible").first
        name_input.wait_for(state="visible", timeout=5000)
        expect(name_input).to_have_value(name, timeout=5000)

    def snapshot_to_active_range(self, name: str, range_name: str) -> None:
        """Snapshot a page or task to the active range via context menu.

        Searches the project tree first (for schematics, line plots, etc.),
        then falls back to the task toolbar (for hardware tasks).

        :param name: Name of the page or task to snapshot.
        :param range_name: Name of the active range (for menu text matching).
        """
        self.pages.expand_active()
        page_item = (
            self.layout.page.locator(".pluto-tree__item").filter(has_text=name).first
        )
        if self.layout.locator_exists(page_item):
            self.ctx_menu.action(page_item, f"Snapshot to {range_name}")
        else:
            self.layout.show_resource_toolbar("Tasks")
            task_item = (
                self.layout.page.locator(".pluto-list__item")
                .filter(has_text=name)
                .first
            )
            task_item.wait_for(state="visible", timeout=5000)
            self.ctx_menu.action(task_item, f"Snapshot to {range_name}")
        self.layout.close_left_toolbar()
