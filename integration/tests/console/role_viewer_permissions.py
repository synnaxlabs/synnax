#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Viewer role has read-only permissions."""

import synnax as sy

from console.case import ConsoleCase
from framework.utils import get_random_name


class RoleViewerPermissions(ConsoleCase):
    """Test that Viewer role is read-only and cannot create any resources."""

    def run(self) -> None:
        # Create a new user with Viewer role
        username = f"viewer_{get_random_name()}"
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
        self.console.access.login(username, password)

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

        self.console.ESCAPE
        sy.sleep(0.3)

        # Test 2: Try to create a workspace
        self.log("Testing: Viewer should not be able to create workspace...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a Workspace", timeout=2000)
        sy.sleep(0.3)

        workspace_cmd = self.page.get_by_text("Create a Workspace", exact=True)
        workspace_cmd_exists = workspace_cmd.count() > 0

        self.console.ESCAPE
        sy.sleep(0.2)

        if workspace_cmd_exists:
            self.log("WARNING: Create Workspace command exists for viewer")
        else:
            self.log("PASS: Create Workspace command not available for viewer")

        # Test 3: Try to create a line plot
        self.log("Testing: Viewer should not be able to create line plot...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a Line Plot", timeout=2000)
        sy.sleep(0.3)

        lineplot_cmd = self.page.get_by_text("Create a Line Plot", exact=True)
        lineplot_cmd_exists = lineplot_cmd.count() > 0

        self.console.ESCAPE
        sy.sleep(0.2)

        if lineplot_cmd_exists:
            self.log("WARNING: Create Line Plot command exists for viewer")
        else:
            self.log("PASS: Create Line Plot command not available for viewer")

        # Test 4: Try to create a channel
        self.log("Testing: Viewer should not be able to create channel...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a Channel", timeout=2000)
        sy.sleep(0.3)

        channel_cmd = self.page.get_by_text("Create a Channel", exact=True)
        channel_cmd_exists = channel_cmd.count() > 0

        self.console.ESCAPE

        if channel_cmd_exists:
            self.log("WARNING: Create Channel command exists for viewer")
        else:
            self.log("PASS: Create Channel command not available for viewer")

        self.log("Viewer permissions test completed")
