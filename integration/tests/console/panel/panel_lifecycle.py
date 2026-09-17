#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Panel strip lifecycle: create, rename, switch, reorder, and delete panels."""

from console.case import ConsoleCase
from console.schematic.schematic import Schematic
from x import random_name


class PanelLifecycle(ConsoleCase):
    """Exercise the panel selector strip end to end."""

    def run(self) -> None:
        self.suffix = random_name()
        self.alpha = f"Alpha_{self.suffix}"
        self.beta = f"Beta_{self.suffix}"
        self.gamma = f"Gamma_{self.suffix}"
        self.test_rename_panel()
        self.test_create_panel()
        self.test_independent_tabs()
        self.test_reorder_panels()
        self.test_delete_selects_neighbor()
        self.test_delete_last_panel()

    def test_rename_panel(self) -> None:
        """Rename the setup-created panel through the strip context menu."""
        self.log("Renaming the initial panel")
        panels = self.console.panels
        panels.rename("New panel", self.alpha)
        assert panels.names() == [self.alpha]

    def test_create_panel(self) -> None:
        """Create a panel; it appears in the strip and becomes selected."""
        self.log("Creating a second panel")
        panels = self.console.panels
        panels.create()
        panels.rename("New panel", self.beta)
        assert panels.names() == [self.alpha, self.beta]
        assert panels.selected_name() == self.beta, (
            "a created panel should become the selected panel"
        )

    def test_independent_tabs(self) -> None:
        """Each panel keeps its own tabs across switches."""
        self.log("Verifying panels keep independent tabs")
        panels = self.console.panels
        panels.select(self.alpha)
        name = f"panel_page_{self.suffix}"
        page = self.console.pages.create(Schematic, name)
        self._cleanup_pages.append(name)
        tab = self.console.layout.get_tab(name)
        assert tab.is_visible(), "created page should open in the selected panel"

        panels.select(self.beta)
        assert not tab.is_visible(), "another panel should not show the tab"
        panels.select(self.alpha)
        assert tab.is_visible(), "switching back should restore the panel's tabs"
        page.close()

    def test_reorder_panels(self) -> None:
        """Drag a pill along the strip to reorder the panels."""
        self.log("Reordering panels by dragging a pill")
        panels = self.console.panels
        panels.create()
        panels.rename("New panel", self.gamma)
        assert panels.names() == [self.alpha, self.beta, self.gamma]

        panels.reorder(self.gamma, self.alpha)
        names = panels.names()
        assert names.index(self.gamma) < names.index(self.beta), (
            f"dragging {self.gamma} onto {self.alpha} should move it "
            f"ahead of {self.beta}, got {names}"
        )

    def test_delete_selects_neighbor(self) -> None:
        """Deleting the selected panel hands selection to its neighbor."""
        self.log("Deleting the selected panel")
        panels = self.console.panels
        order = panels.names()
        target = panels.selected_name()
        index = order.index(target)
        expected = order[index + 1] if index + 1 < len(order) else order[index - 1]

        panels.delete(target)
        assert target not in panels.names()
        assert panels.selected_name() == expected, (
            f"deleting {target!r} should select its neighbor {expected!r}"
        )

    def test_delete_last_panel(self) -> None:
        """Deleting every panel shows the create affordance."""
        self.log("Deleting the remaining panels")
        panels = self.console.panels
        for name in panels.names():
            panels.delete(name)
        assert panels.names() == []
        create = panels.strip.get_by_text("Create panel")
        self.console.layout.wait_for_visible(create)
        # Leave one panel behind so teardown's page cleanup has a mosaic.
        panels.create()
