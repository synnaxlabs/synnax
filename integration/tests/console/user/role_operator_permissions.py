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
from console.case import ConsoleCase
from console.schematic import Setpoint, Valve
from console.schematic.schematic import Schematic
from x import random_name

F64_CHANNEL = f"operator_perm_f64_{random_name()}"
F64_INDEX = f"operator_perm_f64_idx_{random_name()}"
VALVE_CHANNEL = f"operator_perm_vlv_{random_name()}"
VALVE_INDEX = f"operator_perm_vlv_idx_{random_name()}"
SCHEMATIC_NAME = "operator_perm_schematic"

PASSWORD = "testpassword123"
FIRST_NAME = "Operator"


class RoleOperatorPermissions(ConsoleCase):
    """Operator holds framer and range writes, and retrieve on everything else."""

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
        self.owner_creates_schematic()
        self.log_in_as_operator()
        self.badge_names_the_role()
        self.management_surfaces_are_hidden()
        self.creation_commands_are_hidden()
        self.mosaic_is_static()
        self.operator_can_actuate()

    def owner_creates_schematic(self) -> None:
        """As Owner: build the schematic and leave its tab open.

        An Operator cannot open a tab, so anything they view has to already be
        open in a panel an Engineer or Owner built.
        """
        schematic = self.console.project.create_schematic(SCHEMATIC_NAME)
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

    def log_in_as_operator(self) -> None:
        self._username = f"operator_{random_name()}"
        assert self.console.access.register_user(
            username=self._username,
            password=PASSWORD,
            first_name=FIRST_NAME,
            last_name="Test",
            role_name="Operator",
        ), f"failed to register user {self._username}"
        self.console.access.logout()
        self.console.access.login(username=self._username, password=PASSWORD)
        self.page.get_by_text(FIRST_NAME, exact=True).wait_for(
            state="visible", timeout=10000
        )

    def badge_names_the_role(self) -> None:
        """The badge is the only standing signal a restricted session carries."""
        role = self.console.access.get_current_role()
        assert role == "Operator", f"badge shows role {role!r}, expected 'Operator'"

    def management_surfaces_are_hidden(self) -> None:
        assert not self.console.access.users_toolbar_visible(), (
            "Users toolbar is visible to an Operator; it is gated on user update"
        )

    def creation_commands_are_hidden(self) -> None:
        for command in (
            "Create a project",
            "Create a schematic",
            "Create a line plot",
            "Create a channel",
        ):
            assert not self.console.access.command_available(command), (
                f"{command!r} is offered to an Operator, who cannot create one"
            )

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

    def operator_can_actuate(self) -> None:
        """Schematic control is a framer write, which Operator holds in full."""
        schematic = self.console.project.bind_open_page(Schematic, SCHEMATIC_NAME)

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
