#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Engineer role has full access except user management."""

from console.plot import Plot
from console.schematic.schematic import Schematic
from console.table import Table
from tests.console.user.role_case import RoleCase

SCHEMATIC_NAME = "engineer_perm_schematic"
PLOT_NAME = "engineer_perm_plot"
TABLE_NAME = "engineer_perm_table"


class RoleEngineerPermissions(RoleCase):
    """Engineer holds every action except user, role, and policy writes."""

    role_name = "Engineer"

    def run(self) -> None:
        self.login_as_role()
        self.assert_badge_names_role()
        self.assert_users_toolbar_hidden()
        self.creation_commands_are_offered()
        self.mosaic_is_interactive()
        self.tab_menu_offers_the_writes()
        self.visualizations_are_editable()

    def creation_commands_are_offered(self) -> None:
        for command in (
            "Create project",
            "Create schematic",
            "Create line plot",
            "Create channel",
            "Create range",
        ):
            self.assert_command_available(command)

    def mosaic_is_interactive(self) -> None:
        """An Engineer writes panels, so every structural affordance stays."""
        schematic = self.console.pages.create(Schematic, SCHEMATIC_NAME)
        self._cleanup_pages.append(schematic.page_name)
        assert self.console.layout.tab_is_closable(SCHEMATIC_NAME), (
            "tab withholds its close button from an Engineer, who can write the panel"
        )
        assert not self.console.layout.mosaic_is_static(), (
            "mosaic withholds structural writes from an Engineer"
        )

    def tab_menu_offers_the_writes(self) -> None:
        for option in ("Rename", "Move to panel"):
            assert self.console.layout.tab_menu_has_option(SCHEMATIC_NAME, option), (
                f"tab menu withholds {option!r} from an Engineer, who writes the panel"
            )

    def visualizations_are_editable(self) -> None:
        """An Engineer writes plots and tables, so both offer their editing surface."""
        plot = self.console.pages.create(Plot, PLOT_NAME)
        self._cleanup_pages.append(plot.page_name)
        self.console.layout.show_visualization_toolbar()
        self.console.layout.get_by_text("Axes", exact=True).wait_for(
            state="visible", timeout=10000
        )
        table = self.console.pages.create(Table, TABLE_NAME)
        self._cleanup_pages.append(table.page_name)
        # A fresh table starts editable, so the toolbar shows the cell form when
        # a cell is selected and the selection prompt otherwise.
        editing_surface = self.console.layout.get_by_text("Variant", exact=True).or_(
            self.console.layout.get_by_text("No cell selected")
        )
        editing_surface.first.wait_for(state="visible", timeout=10000)
        self.console.layout.hide_visualization_toolbar()
