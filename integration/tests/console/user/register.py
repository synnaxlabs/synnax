#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test user registration and management via Console UI."""

from console.case import ConsoleCase
from x import random_name

PASSWORD = "testpassword123"


class UserRegister(ConsoleCase):
    """Test registering a new user and user management operations."""

    def run(self) -> None:
        self.username = ""
        self.name = ""
        self.test_register_user()
        self.test_change_role()
        self.test_role_change_affects_permissions()
        self.test_rename_user()
        self.test_change_username_and_login()
        self.test_delete_user()

    def test_register_user(self) -> None:
        """Register a new user and verify the Users page shows their role."""
        self.username = f"testuser_{random_name()}"
        self.name = self.username
        role_name = "Operator"

        self.log(f"Registering user: {self.username}")

        self.console.access.register_user(
            username=self.username,
            password=PASSWORD,
            role_name=role_name,
        )

        self.log("User registered, verifying in UI...")
        self.console.access.wait_for_user_role(self.username, role_name=role_name)

    def test_change_role(self) -> None:
        """Change a user's role and verify the Users page shows it."""
        self.log(f"Changing role for {self.username}: Operator -> Engineer")

        self.console.access.assign_role_to_user(
            username=self.username, role_name="Engineer"
        )

        self.console.access.wait_for_user_role(self.username, role_name="Engineer")

    def test_role_change_affects_permissions(self) -> None:
        """Log in as the user (now Engineer) and verify they gained Engineer
        permissions.
        """
        self.log("Logging in as changed user to verify permissions")
        self.console.access.logout()
        self.console.access.login(username=self.username, password=PASSWORD)

        self.log("Testing: Engineer can create a project")
        self.console.layout.command_palette("Create project")
        self.console.layout.close_modal(self.console.layout.MODAL_SELECTOR)

        self.log("Logging back in as admin")
        self.console.access.logout()
        self.console.access.login(username="synnax", password="seldon")

    def test_rename_user(self) -> None:
        """Rename a user via the context menu on the Users page."""
        new_name = f"renamed_{random_name()}"
        self.log(f"Renaming user: {self.name} -> {new_name}")

        self.console.access.rename_user(name=self.name, new_name=new_name)

        item = self.console.access.find_user_item(new_name)
        assert item is not None, f"Renamed user {new_name} not found on the Users page"

        self.name = new_name

    def test_change_username_and_login(self) -> None:
        """Change a user's username, then log in with the new username."""
        new_username = f"logintest_{random_name()}"
        self.log(f"Changing username: {self.username} -> {new_username}")

        user = self.client.users.retrieve(username=self.username)
        self.client.users.change_username(user.key, new_username)
        self.username = new_username

        self.log("Logging out and logging in with new username")
        self.console.access.logout()
        self.console.access.login(username=self.username, password=PASSWORD)

        self.log("Login succeeded, logging back in as admin")
        self.console.access.logout()
        self.console.access.login(username="synnax", password="seldon")

    def test_delete_user(self) -> None:
        """Delete a user via the context menu on the Users page."""
        username = f"deluser_{random_name()}"
        self.log(f"Registering user to delete: {username}")

        self.console.access.register_user(
            username=username,
            password=PASSWORD,
            role_name="Operator",
        )

        self.log(f"Deleting user: {username}")
        self.console.access.delete_user(username)
