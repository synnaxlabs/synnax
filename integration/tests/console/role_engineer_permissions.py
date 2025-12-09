#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Engineer role has full access except user management."""

import uuid

import synnax as sy

from console.case import ConsoleCase


class RoleEngineerPermissions(ConsoleCase):
    """Test that Engineer can create resources but cannot manage users."""

    def run(self) -> None:
        # Create a new user with Engineer role
        username = f"engineer_{uuid.uuid4().hex[:8]}"
        password = "testpassword123"
        first_name = "Engineer"
        last_name = "Test"
        role_name = "Engineer"

        self.log(f"Registering engineer user: {username}")

        success = self.console.access.register_user(
            username=username,
            password=password,
            first_name=first_name,
            last_name=last_name,
            role_name=role_name,
        )
        assert success, f"Failed to register user {username}"

        # Log out and log in as the engineer
        self.log("Logging out and logging in as engineer...")
        self.console.access.logout()
        self.console.access.login(username, password)

        # Verify logged in as engineer
        user_badge = self.page.get_by_text(first_name, exact=True)
        user_badge.wait_for(state="visible", timeout=10000)
        self.log(f"Logged in as engineer: {first_name}")

        # Test 1: Users toolbar should be hidden (Engineer can view but not edit users)
        self.log("Testing: Users toolbar should be hidden for engineer...")
        self.page.keyboard.press("u")
        sy.sleep(0.5)

        role_elements = self.page.locator("div[id^='role:']")
        users_visible = role_elements.count() > 0 and role_elements.first.is_visible()

        if users_visible:
            self.log("WARNING: Users toolbar is visible to engineer (unexpected)")
        else:
            self.log("PASS: Users toolbar is hidden for engineer")

        self.console.ESCAPE
        sy.sleep(0.3)

        # Test 2: Engineer SHOULD be able to create workspace
        self.log("Testing: Engineer should be able to create workspace...")
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
            self.log("PASS: Create Workspace command available for engineer")
        else:
            self.log("FAIL: Create Workspace command not available for engineer")
            assert False, "Engineer should be able to create workspace"

        # Test 3: Engineer SHOULD be able to create schematic
        self.log("Testing: Engineer should be able to create schematic...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a Schematic", timeout=2000)
        sy.sleep(0.3)

        schematic_cmd = self.page.get_by_text("Create a Schematic", exact=True)
        schematic_cmd_exists = schematic_cmd.count() > 0

        self.console.ESCAPE
        sy.sleep(0.2)

        if schematic_cmd_exists:
            self.log("PASS: Create Schematic command available for engineer")
        else:
            self.log("FAIL: Create Schematic command not available for engineer")
            assert False, "Engineer should be able to create schematic"

        # Test 4: Engineer SHOULD be able to create channel
        self.log("Testing: Engineer should be able to create channel...")
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
            self.log("PASS: Create Channel command available for engineer")
        else:
            self.log("FAIL: Create Channel command not available for engineer")
            assert False, "Engineer should be able to create channel"

        self.log("Engineer permissions test completed")
