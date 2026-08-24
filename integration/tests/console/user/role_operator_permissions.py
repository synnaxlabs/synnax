#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Operator can actuate controls but cannot restructure or configure."""

import synnax as sy
from console.plot import Plot
from console.schematic import Setpoint, Valve
from console.schematic.schematic import Schematic
from console.table import Table
from tests.console.user.role_case import RoleCase
from x import random_name

F64_CHANNEL = f"operator_perm_f64_{random_name()}"
F64_INDEX = f"operator_perm_f64_idx_{random_name()}"
VALVE_CHANNEL = f"operator_perm_vlv_{random_name()}"
VALVE_INDEX = f"operator_perm_vlv_idx_{random_name()}"
SCHEMATIC_NAME = "operator_perm_schematic"
PLOT_NAME = "operator_perm_plot"
TABLE_NAME = "operator_perm_table"


class RoleOperatorPermissions(RoleCase):
    """Operator holds framer and range writes, and retrieve on everything else."""

    role_name = "Operator"

    def setup(self) -> None:
        f64_idx = self.client.channels.create(
            name=F64_INDEX,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=F64_CHANNEL,
            data_type=sy.DataType.FLOAT64,
            index=f64_idx.key,
            retrieve_if_name_exists=True,
        )
        vlv_idx = self.client.channels.create(
            name=VALVE_INDEX,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=VALVE_CHANNEL,
            data_type=sy.DataType.UINT8,
            index=vlv_idx.key,
            retrieve_if_name_exists=True,
        )
        self.subscribe([F64_CHANNEL, VALVE_CHANNEL])
        super().setup()

    def run(self) -> None:
        self.owner_creates_pages()
        self.login_as_role()
        self.assert_badge_names_role()
        self.assert_users_toolbar_hidden()
        self.creation_commands_are_hidden()
        self.mosaic_is_static()
        self.tab_menu_offers_no_writes()
        self.visualizations_are_read_only()
        self.operator_can_actuate()

    def owner_creates_pages(self) -> None:
        """As Owner: build the pages and leave their tabs open.

        An Operator cannot open a tab, so anything they view has to already be
        open in a panel an Engineer or Owner built.
        """
        table = self.console.pages.create(Table, TABLE_NAME)
        self._cleanup_pages.append(table.page_name)
        plot = self.console.pages.create(Plot, PLOT_NAME)
        self._cleanup_pages.append(plot.page_name)
        schematic = self.console.pages.create(Schematic, SCHEMATIC_NAME)
        self._cleanup_pages.append(schematic.page_name)

        setpoint = schematic.create_symbol(
            Setpoint(label=F64_CHANNEL, channel_name=F64_CHANNEL)
        )
        setpoint.move(delta_x=-200, delta_y=0)

        valve = schematic.create_symbol(
            Valve(
                label=VALVE_CHANNEL,
                state_channel=VALVE_CHANNEL,
                command_channel=VALVE_CHANNEL,
            )
        )
        valve.move(delta_x=200, delta_y=0)

    def creation_commands_are_hidden(self) -> None:
        for command in (
            "Create project",
            "Create schematic",
            "Create line plot",
            "Create channel",
        ):
            self.assert_command_hidden(command)

    def mosaic_is_static(self) -> None:
        """An Operator cannot write panels, so no structural affordance appears."""
        self.console.layout.get_read_only_tab(SCHEMATIC_NAME).wait_for(
            state="visible", timeout=10000
        )
        assert not self.console.layout.tab_is_closable(SCHEMATIC_NAME), (
            "tab offers a close button to an Operator, who cannot write the panel"
        )
        assert self.console.layout.mosaic_is_static(), (
            "mosaic offers a structural write to an Operator"
        )

    def tab_menu_offers_no_writes(self) -> None:
        """Renaming or moving a tab rewrites the panel, which an Operator cannot."""
        for option in ("Rename", "Move to panel", "Move to new window"):
            assert not self.console.layout.tab_menu_has_option(
                SCHEMATIC_NAME, option
            ), f"tab menu offers {option!r} to an Operator, who cannot write the panel"

    def visualizations_are_read_only(self) -> None:
        """Editing a plot or a table is a write on it, which an Operator cannot."""
        self.console.layout.select_tab(PLOT_NAME)
        self.console.layout.show_visualization_toolbar()
        assert self.console.layout.get_by_text("Axes", exact=True).count() == 0, (
            "line plot offers its editing tabs to an Operator, who cannot write it"
        )
        self.console.layout.select_tab(TABLE_NAME)
        self.console.layout.get_by_text(f"{TABLE_NAME} is not editable").wait_for(
            state="visible", timeout=10000
        )
        self.console.layout.hide_visualization_toolbar()
        self.console.layout.select_tab(SCHEMATIC_NAME)

    def operator_can_actuate(self) -> None:
        """Schematic control is a framer write, which Operator holds in full."""
        schematic = self.console.pages.bind_open(Schematic, SCHEMATIC_NAME)

        self.log("Testing: Operator sends an f64 setpoint")
        setpoint = schematic.find_symbol(
            Setpoint(label=F64_CHANNEL, channel_name=F64_CHANNEL)
        )
        setpoint.set_value(7.89)
        self.wait_for_eq(F64_CHANNEL, 7.89)

        self.log("Testing: Operator opens and closes a valve")
        valve = schematic.find_symbol(
            Valve(
                label=VALVE_CHANNEL,
                state_channel=VALVE_CHANNEL,
                command_channel=VALVE_CHANNEL,
            )
        )
        valve.press()
        self.wait_for_eq(VALVE_CHANNEL, 1)
        valve.press()
        self.wait_for_eq(VALVE_CHANNEL, 0)
