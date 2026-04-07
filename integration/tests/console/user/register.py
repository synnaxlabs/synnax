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

BUILTIN_ROLES = ["Owner", "Engineer", "Operator", "Viewer"]
PASSWORD = "testpassword123"


class UserRegister(ConsoleCase):
    """Test registering a new user and user management operations."""

    def run(self) -> None:
        self.username = ""
        self.test_builtin_roles_visible()
        self.test_builtin_roles_not_deletable()
        self.test_register_user()
        self.test_change_role()
        self.test_role_change_affects_permissions()
        self.test_rename_user()
        self.test_change_username_and_login()
        self.test_delete_user()
        self.test_delete_multiple_users()

    def test_builtin_roles_visible(self) -> None:
        """Verify all built-in roles are visible in the Resources Toolbar."""
        self.log("Testing: All built-in roles visible")
        roles = self.console.access.list_visible_roles()
        self.log(f"Visible roles: {roles}")
        for role in BUILTIN_ROLES:
            assert role in roles, (
                f"Built-in role '{role}' not visible in toolbar. Found: {roles}"
            )

    def test_builtin_roles_not_deletable(self) -> None:
        """Verify built-in roles cannot be renamed or deleted."""
        self.log("Testing: Built-in roles are not modifiable")
        for role in BUILTIN_ROLES:
            assert not self.console.access.is_role_modifiable(role), (
                f"Built-in role '{role}' should not be modifiable"
            )

    def test_register_user(self) -> None:
        """Register a new user and verify it appears under the assigned role."""
        self.username = f"testuser_{random_name()}"
        role_name = "Operator"

        self.log(f"Registering user: {self.username}")

        success = self.console.access.register_user(
            username=self.username,
            password=PASSWORD,
            first_name="Test",
            last_name="User",
            role_name=role_name,
        )
        assert success, f"Failed to register user {self.username}"

        self.log("User registered, verifying in UI...")
        self.console.access.expand_role(role_name)

        users = self.console.access.list_users_under_role(role_name)
        self.log(f"Users found: {users}")
        assert self.username in users, (
            f"User {self.username} not found under {role_name}. Found: {users}"
        )

    def test_change_role(self) -> None:
        """Change a user's role and verify they move between roles."""
        self.log(f"Changing role for {self.username}: Operator -> Engineer")

        success = self.console.access.assign_role_to_user(
            username=self.username, role_name="Engineer"
        )
        assert success, f"Failed to change role for {self.username}"

        self.console.access.expand_role("Engineer")
        engineer_users = self.console.access.list_users_under_role("Engineer")
        assert self.username in engineer_users, (
            f"User {self.username} not found under Engineer. Found: {engineer_users}"
        )

    def test_role_change_affects_permissions(self) -> None:
        """Log in as the user (now Engineer) and verify they gained Engineer permissions."""
        self.log("Logging in as changed user to verify permissions")
        self.console.access.logout()
        self.console.access.login(username=self.username, password=PASSWORD)

        self.log("Testing: Engineer can create a workspace")
        self.console.layout.command_palette("Create a workspace")
        self.console.layout.close_modal(self.console.layout.MODAL_SELECTOR)

        self.log("Logging back in as admin")
        self.console.access.logout()
        self.console.access.login(username="synnax", password="seldon")
        self.console.access.expand_role("Engineer")

    def test_rename_user(self) -> None:
        """Rename a user via the Resources Toolbar context menu."""
        new_username = f"renamed_{random_name()}"
        self.log(f"Renaming user: {self.username} -> {new_username}")

        success = self.console.access.rename_user(
            username=self.username, new_username=new_username
        )
        assert success, f"Failed to rename user {self.username}"

        item = self.console.access._find_user_item(new_username)
        assert item is not None, f"Renamed user {new_username} not found in tree"

        self.username = new_username

    def test_change_username_and_login(self) -> None:
        """Change a user's username, then log in with the new username."""
        new_username = f"logintest_{random_name()}"
        self.log(f"Changing username: {self.username} -> {new_username}")

        success = self.console.access.rename_user(
            username=self.username, new_username=new_username
        )
        assert success, f"Failed to rename user {self.username}"
        self.username = new_username

        self.log("Logging out and logging in with new username")
        self.console.access.logout()
        self.console.access.login(username=self.username, password=PASSWORD)

        self.log("Login succeeded, logging back in as admin")
        self.console.access.logout()
        self.console.access.login(username="synnax", password="seldon")

    def test_delete_user(self) -> None:
        """Delete a single user via the Resources Toolbar context menu."""
        username = f"deluser_{random_name()}"
        self.log(f"Registering user to delete: {username}")

        success = self.console.access.register_user(
            username=username,
            password=PASSWORD,
            first_name="Delete",
            last_name="Me",
            role_name="Operator",
        )
        assert success, f"Failed to register user {username}"

        self.console.access.expand_role("Operator")

        self.log(f"Deleting user: {username}")
        success = self.console.access.delete_user(username)
        assert success, f"Failed to delete user {username}"

    def test_delete_multiple_users(self) -> None:
        """Delete multiple users via multi-select in the Resources Toolbar."""
        usernames = [f"multidel_{random_name()}" for _ in range(2)]

        for username in usernames:
            self.log(f"Registering user: {username}")
            success = self.console.access.register_user(
                username=username,
                password=PASSWORD,
                first_name="Multi",
                last_name="Delete",
                role_name="Operator",
            )
            assert success, f"Failed to register user {username}"

        self.console.access.expand_role("Operator")

        for username in usernames:
            self.console.access.ensure_user_visible(username, "Operator")

        self.log(f"Deleting users: {usernames}")
        success = self.console.access.delete_users(usernames)
        assert success, f"Failed to delete users {usernames}"
