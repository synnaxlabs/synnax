#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Viewer role has read-only permissions and cannot actuate controls."""

import synnax as sy
from console.case import ConsoleCase
from console.schematic import Setpoint, Valve
from console.schematic.schematic import Schematic
from x import random_name

F64_CHANNEL = f"viewer_perm_f64_{random_name()}"
F64_INDEX = f"viewer_perm_f64_idx_{random_name()}"
VALVE_CHANNEL = f"viewer_perm_vlv_{random_name()}"
VALVE_INDEX = f"viewer_perm_vlv_idx_{random_name()}"
SCHEMATIC_NAME = "viewer_perm_schematic"

PASSWORD = "testpassword123"
FIRST_NAME = "Viewer"


class RoleViewerPermissions(ConsoleCase):
    """Viewer holds retrieve on everything and no write anywhere."""

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
        self.owner_creates_and_actuates()
        self.log_in_as_viewer()
        self.badge_names_the_role()
        self.management_surfaces_are_hidden()
        self.creation_commands_are_hidden()
        self.mosaic_is_static()
        self.viewer_cannot_actuate()

    def owner_creates_and_actuates(self) -> None:
        """As Owner: build the schematic, prove control works, leave the tab open."""
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

        self.log("Testing: Owner sends f64 setpoint")
        setpoint.set_value(1.23)
        self.wait_for_eq(F64_CHANNEL, 1.23)

        self.log("Testing: Owner opens valve")
        valve.press()
        self.wait_for_eq(VALVE_CHANNEL, 1)
        valve.press()
        self.wait_for_eq(VALVE_CHANNEL, 0)

    def log_in_as_viewer(self) -> None:
        self._username = f"viewer_{random_name()}"
        assert self.console.access.register_user(
            username=self._username,
            password=PASSWORD,
            first_name=FIRST_NAME,
            last_name="Test",
            role_name="Viewer",
        ), f"failed to register user {self._username}"
        self.console.access.logout()
        self.console.access.login(username=self._username, password=PASSWORD)
        self.page.get_by_text(FIRST_NAME, exact=True).wait_for(
            state="visible", timeout=10000
        )

    def badge_names_the_role(self) -> None:
        role = self.console.access.get_current_role()
        assert role == "Viewer", f"badge shows role {role!r}, expected 'Viewer'"

    def management_surfaces_are_hidden(self) -> None:
        assert not self.console.access.users_toolbar_visible(), (
            "Users toolbar is visible to a Viewer; it is gated on user update"
        )

    def creation_commands_are_hidden(self) -> None:
        for command in (
            "Create a project",
            "Create a line plot",
            "Create a channel",
            "Create a schematic",
            "Define a range",
        ):
            assert not self.console.access.command_available(command), (
                f"{command!r} is offered to a Viewer, who cannot create one"
            )

    def mosaic_is_static(self) -> None:
        self.console.layout.get_read_only_tab(SCHEMATIC_NAME).wait_for(
            state="visible", timeout=10000
        )
        assert not self.console.layout.tab_is_closable(SCHEMATIC_NAME), (
            "tab offers a close button to a Viewer, who cannot write the panel"
        )
        assert self.console.layout.mosaic_is_static(), (
            "mosaic offers a structural write to a Viewer"
        )

    def viewer_cannot_actuate(self) -> None:
        """A Viewer holds no framer create, so control affordances are gone."""
        schematic = self.console.project.bind_open_page(Schematic, SCHEMATIC_NAME)

        assert not schematic.has_control_toggle(), (
            "schematic offers control acquisition to a Viewer, who cannot write framer"
        )

        self.log("Testing: Viewer tries to send f64 value")
        setpoint = schematic.find_symbol(
            Setpoint(label=F64_CHANNEL, channel_name=F64_CHANNEL)
        )
        setpoint.set_value(4.56)
        self.wait_for_eq(F64_CHANNEL, 1.23, timeout=0)

        self.log("Testing: Viewer tries to open valve")
        valve = schematic.find_symbol(
            Valve(
                label=VALVE_CHANNEL,
                state_channel=VALVE_CHANNEL,
                command_channel=VALVE_CHANNEL,
            )
        )
        valve.press()
        self.wait_for_eq(VALVE_CHANNEL, 0, timeout=0)
