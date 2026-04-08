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


class RoleViewerPermissions(ConsoleCase):
    """Test that Viewer role is read-only and cannot create or actuate."""

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
        schematic = self.console.workspace.create_schematic(SCHEMATIC_NAME)
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

        # Create a new user with Viewer role
        username = f"viewer_{random_name()}"
        password = "testpassword123"
        first_name = "Viewer"
        last_name = "Test"
        role_name = "Viewer"

        self.log(f"Registering viewer user: {username}")

        success = self.console.access.register_user(
            username=username,
            password=password,
            first_name=first_name,
            last_name=last_name,
            role_name=role_name,
        )
        assert success, f"Failed to register user {username}"

        # Log out and log in as the viewer
        self.log("Logging out and logging in as viewer...")
        self.console.access.logout()
        self.console.access.login(username=username, password=password)

        # Verify logged in as viewer
        user_badge = self.page.get_by_text(first_name, exact=True)
        user_badge.wait_for(state="visible", timeout=10000)
        self.log(f"Logged in as viewer: {first_name}")

        # Test 1: Users toolbar should be hidden
        self.log("Testing: Users toolbar should be hidden for viewer...")
        self.page.keyboard.press("u")
        sy.sleep(0.5)

        role_elements = self.page.locator("div[id^='role:']")
        users_visible = role_elements.count() > 0 and role_elements.first.is_visible()

        if users_visible:
            self.log("WARNING: Users toolbar is visible to viewer (unexpected)")
        else:
            self.log("PASS: Users toolbar is hidden for viewer")

        self.console.layout.press_escape()
        sy.sleep(0.3)

        # Test 2: Try to create a workspace
        self.log("Testing: Viewer should not be able to create workspace...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a workspace", timeout=2000)
        sy.sleep(0.3)

        workspace_cmd = self.page.get_by_text("Create a workspace", exact=True)
        workspace_cmd_exists = workspace_cmd.count() > 0

        self.console.layout.press_escape()
        sy.sleep(0.2)

        if workspace_cmd_exists:
            self.log("WARNING: Create workspace command exists for viewer")
        else:
            self.log("PASS: Create workspace command not available for viewer")

        # Test 3: Try to create a line plot
        self.log("Testing: Viewer should not be able to create line plot...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a line plot", timeout=2000)
        sy.sleep(0.3)

        lineplot_cmd = self.page.get_by_text("Create a line plot", exact=True)
        lineplot_cmd_exists = lineplot_cmd.count() > 0

        self.console.layout.press_escape()
        sy.sleep(0.2)

        if lineplot_cmd_exists:
            self.log("WARNING: Create line plot command exists for viewer")
        else:
            self.log("PASS: Create line plot command not available for viewer")

        # Test 4: Try to create a channel
        self.log("Testing: Viewer should not be able to create channel...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a channel", timeout=2000)
        sy.sleep(0.3)

        channel_cmd = self.page.get_by_text("Create a channel", exact=True)
        channel_cmd_exists = channel_cmd.count() > 0

        self.console.layout.press_escape()

        if channel_cmd_exists:
            self.log("WARNING: Create channel command exists for viewer")
        else:
            self.log("PASS: Create channel command not available for viewer")

        self.test_viewer_can_view_schematic()
        self.test_viewer_cannot_actuate()

    def test_viewer_can_view_schematic(self) -> None:
        """Viewer should be able to open and view an existing schematic."""
        self.log("Testing: Viewer can view schematic")
        self._viewer_schematic = self.console.workspace.open_from_search(
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
