#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase
from console.project import PageType
from x import random_name


class Tombstone(ConsoleCase):
    """
    Deleting an open page tombstones its tab in place for every visualization
    type; Restore heals the view and brings the page back, Close removes the tab.
    """

    def run(self) -> None:
        suffix = random_name()
        pages: list[tuple[PageType, str]] = [
            ("Line Plot", f"TombPlot_{suffix}"),
            ("Schematic", f"TombSch_{suffix}"),
            ("Log", f"TombLog_{suffix}"),
            ("Table", f"TombTable_{suffix}"),
        ]
        for page_type, page_name in pages:
            self.test_tombstone_lifecycle(page_type, page_name)

    def test_tombstone_lifecycle(self, page_type: PageType, page_name: str) -> None:
        """Delete an open page, restore it from the tombstone, then delete it
        again and dismiss the tombstone with Close."""
        console = self.console

        self.log(f"({page_type}) Creating page")
        console.project.create_page(page_type, page_name)
        self._cleanup_pages.append(page_name)

        self.log(f"({page_type}) Deleting while the tab is open")
        console.project.delete_page(page_name)

        # The tab is never closed out from under the user: it stays open and
        # renders the tombstone in place of content.
        tombstone = console.layout.get_tombstone(page_name)
        tombstone.wait_for(state="visible", timeout=5000)
        tab = console.layout.get_tab(page_name)
        assert tab.is_visible(), f"Tab '{page_name}' should stay open after delete"

        self.log(f"({page_type}) Restoring from the tombstone")
        console.layout.restore_tombstone(page_name)
        tombstone.wait_for(state="hidden", timeout=5000)
        assert console.project.page_exists(page_name), (
            f"Restored page '{page_name}' should be back in the project tree"
        )
        console.layout.close_left_toolbar()

        self.log(f"({page_type}) Deleting again and dismissing with Close")
        console.project.delete_page(page_name)
        console.layout.get_tombstone(page_name).wait_for(state="visible", timeout=5000)
        console.layout.close_tombstone(page_name)
        console.layout.get_tab(page_name).wait_for(state="detached", timeout=5000)
        self._cleanup_pages.remove(page_name)
