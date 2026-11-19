#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from playwright.sync_api import expect

from console.layout import LayoutClient
from console.range.explorer import Explorer
from console.range.overview import Overview
from console.range.surface import CREATE_MODAL_SELECTOR, Surface
from console.range.toolbar import Toolbar


class Client(Surface):
    """Console ranges client. Surface-specific operations live on
    ``toolbar``, ``explorer``, and ``overview``."""

    def __init__(self, layout: LayoutClient):
        super().__init__(layout)
        self.toolbar = Toolbar(layout)
        self.explorer = Explorer(layout=layout, toolbar=self.toolbar)
        self.overview = Overview(
            layout=layout, toolbar=self.toolbar, explorer=self.explorer
        )

    def create(
        self,
        name: str,
        *,
        parent: str | None = None,
        labels: list[str] | None = None,
        stage: str | None = None,
    ) -> None:
        """Create a new range via the command palette.

        :param name: The name for the new range.
        :param parent: Optional parent range name to set.
        :param labels: Optional list of label names to add.
        :param stage: Optional stage to set ("To do", "In progress",
            "Completed").
        """
        self.layout.command_palette("Create range")
        modal = self.layout.page.locator(CREATE_MODAL_SELECTOR)
        modal.wait_for(state="visible", timeout=5000)
        self.fill_create_modal(name, parent=parent, labels=labels, stage=stage)

    def favorite(self, name: str) -> None:
        """Favorite a range from the explorer (idempotent).

        The overview no longer carries a favorite control; favoriting happens
        on list items in the explorer and toolbar.

        :param name: The name of the range to favorite.
        """
        self.explorer.open()
        self.explorer.favorite(name)
        self.explorer.views.close_tab()

    def open_from_search(self, name: str) -> None:
        """Open a range overview by searching its name in the command palette.

        :param name: Name of the range to search for and open.
        """
        self.layout.search_palette(name)
        name_input = self.layout.page.locator("input[placeholder='Name']").visible.first
        name_input.wait_for(state="visible", timeout=5000)
        expect(name_input).to_have_value(name, timeout=5000)
