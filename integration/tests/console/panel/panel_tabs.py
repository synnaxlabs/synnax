#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Moving tabs between panels, minting panels from tabs, and reload restore."""

from console.case import ConsoleCase
from console.log import Log
from console.schematic.schematic import Schematic
from console.table import Table
from x import random_name


class PanelTabs(ConsoleCase):
    """Move tabs across panels through the picker, drags, and reloads."""

    def run(self) -> None:
        self.suffix = random_name()
        self.alpha = f"Alpha_{self.suffix}"
        self.beta = f"Beta_{self.suffix}"
        panels = self.console.panels
        panels.rename("New panel", self.alpha)
        panels.create()
        panels.rename("New panel", self.beta)

        self.test_move_tab_via_picker()
        self.test_move_tab_to_new_panel()
        self.test_drag_tab_to_pill()
        self.test_drag_tab_to_create_button()
        self.test_reload_restores_panels()

    def test_move_tab_via_picker(self) -> None:
        """Move a tab to another panel through the tab menu's picker."""
        self.log("Moving a tab through the move picker")
        panels = self.console.panels
        panels.select(self.alpha)
        self.table_name = f"panel_table_{self.suffix}"
        self.console.pages.create(Table, self.table_name)
        self._cleanup_pages.append(self.table_name)

        panels.move_tab_to_panel(self.table_name, self.beta)
        panels.select(self.alpha)
        assert not self.console.layout.get_tab(self.table_name).is_visible(), (
            "the moved tab should leave its origin panel"
        )
        panels.select(self.beta)
        self.console.layout.wait_for_tab(self.table_name)

    def test_move_tab_to_new_panel(self) -> None:
        """The picker's "New panel" entry mints a panel named after the tab."""
        self.log("Moving a tab to a minted panel")
        self.console.panels.move_tab_to_panel(self.table_name, None)

    def test_drag_tab_to_pill(self) -> None:
        """Drag a mosaic tab onto a strip pill to move it there."""
        self.log("Dragging a tab onto a panel pill")
        panels = self.console.panels
        self.log_name = f"panel_log_{self.suffix}"
        self.console.pages.create(Log, self.log_name)
        self._cleanup_pages.append(self.log_name)

        panels.drag_tab_to_pill(self.log_name, self.alpha)

    def test_drag_tab_to_create_button(self) -> None:
        """Drag a mosaic tab onto the create button to mint a panel for it."""
        self.log("Dragging a tab onto the create button")
        panels = self.console.panels
        self.schematic_name = f"panel_schematic_{self.suffix}"
        self.console.pages.create(Schematic, self.schematic_name)
        self._cleanup_pages.append(self.schematic_name)

        panels.drag_tab_to_create_button(self.schematic_name)

    def test_reload_restores_panels(self) -> None:
        """Panels, their order, and their tabs survive a Console reload."""
        self.log("Reloading the Console")
        panels = self.console.panels
        before = panels.names()
        panels.reload_console()
        assert panels.names() == before, (
            f"panel strip should restore after reload: {before} != {panels.names()}"
        )
        panels.select(self.schematic_name)
        self.console.layout.wait_for_tab(self.schematic_name)
