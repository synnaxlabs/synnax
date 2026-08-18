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
from console.schematic import Setpoint, Valve
from console.schematic.schematic import Schematic
from tests.console.user.role_case import RoleCase
from x import random_name

F64_CHANNEL = f"viewer_perm_f64_{random_name()}"
F64_INDEX = f"viewer_perm_f64_idx_{random_name()}"
VALVE_CHANNEL = f"viewer_perm_vlv_{random_name()}"
VALVE_INDEX = f"viewer_perm_vlv_idx_{random_name()}"
SCHEMATIC_NAME = "viewer_perm_schematic"


class RoleViewerPermissions(RoleCase):
    """Test that Viewer role is read-only and cannot create or actuate."""

    role_name = "Viewer"

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

    def test_owner_creates_schematic(self) -> None:
        """As Owner: create a schematic with f64 setpoint and boolean button."""
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

        self.log("Testing: Owner sends f64 setpoint")
        setpoint.set_value(1.23)
        self.wait_for_eq(F64_CHANNEL, 1.23)

        self.log("Testing: Owner opens valve")
        valve.press()
        self.wait_for_eq(VALVE_CHANNEL, 1)
        valve.press()
        self.wait_for_eq(VALVE_CHANNEL, 0)

    def run(self) -> None:
        self.test_owner_creates_schematic()

        self.login_as_role()
        self.assert_users_toolbar_hidden()
        self.assert_command_hidden("Create project")
        self.assert_command_hidden("Create line plot")
        self.assert_command_hidden("Create channel")

        self.test_viewer_can_view_schematic()
        self.test_viewer_cannot_actuate()

    def test_viewer_can_view_schematic(self) -> None:
        """Viewer should be able to open and view an existing schematic."""
        self.log("Testing: Viewer can view schematic")
        self._viewer_schematic = self.console.pages.open_from_search(
            Schematic, SCHEMATIC_NAME
        )

    def test_viewer_cannot_actuate(self) -> None:
        """Viewer should not be able to send commands via schematic controls."""
        schematic = self._viewer_schematic

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
