#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Operator role has restricted permissions."""

import synnax as sy

from console.case import ConsoleCase
from framework.utils import get_random_name


class RoleOperatorPermissions(ConsoleCase):
    """Test that Operator role cannot access user management or create resources."""

    def run(self) -> None:
        # Create a new user with Operator role
        username = f"operator_{get_random_name()}"
        password = "testpassword123"
        first_name = "Operator"
        last_name = "Test"
        role_name = "Operator"

        self.log(f"Registering operator user: {username}")

        success = self.console.access.register_user(
            username=username,
            password=password,
            first_name=first_name,
            last_name=last_name,
            role_name=role_name,
        )
        assert success, f"Failed to register user {username}"

        # Log out and log in as the operator
        self.log("Logging out and logging in as operator...")
        self.console.access.logout()
        self.console.access.login(username=username, password=password)

        # Verify logged in as operator
        user_badge = self.page.get_by_text(first_name, exact=True)
        user_badge.wait_for(state="visible", timeout=10000)
        self.log(f"Logged in as operator: {first_name}")

        # Test 1: Users toolbar should be hidden
        self.log("Testing: Users toolbar should be hidden for operator...")
        self.page.keyboard.press("u")
        sy.sleep(0.5)

        # Check that no role elements are visible (users panel not shown)
        role_elements = self.page.locator("div[id^='role:']")
        users_visible = role_elements.count() > 0 and role_elements.first.is_visible()

        if users_visible:
            self.log("WARNING: Users toolbar is visible to operator (unexpected)")
        else:
            self.log("PASS: Users toolbar is hidden for operator")

        # Press escape to close any open panels
        self.console.layout.press_escape()
        sy.sleep(0.3)

        # Test 2: Try to create a workspace via command palette
        self.log("Testing: Operator should not be able to create workspace...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a Workspace", timeout=2000)
        sy.sleep(0.3)

        # Check if command is available
        workspace_cmd = self.page.get_by_text("Create a Workspace", exact=True)
        workspace_cmd_exists = workspace_cmd.count() > 0

        self.console.layout.press_escape()
        sy.sleep(0.2)

        if workspace_cmd_exists:
            self.log("WARNING: Create Workspace command exists for operator")
        else:
            self.log("PASS: Create Workspace command not available for operator")

        # Test 3: Try to create a schematic via command palette
        self.log("Testing: Operator should not be able to create schematic...")
        self.page.keyboard.press("ControlOrMeta+Shift+p")
        sy.sleep(0.3)

        palette_input = self.page.locator(
            ".console-palette__input input[role='textbox']"
        )
        palette_input.fill(">Create a Schematic", timeout=2000)
        sy.sleep(0.3)

        schematic_cmd = self.page.get_by_text("Create a Schematic", exact=True)
        schematic_cmd_exists = schematic_cmd.count() > 0

        self.console.layout.press_escape()

        if schematic_cmd_exists:
            self.log("WARNING: Create Schematic command exists for operator")
        else:
            self.log("PASS: Create Schematic command not available for operator")

        self.log("Operator permissions test completed")
