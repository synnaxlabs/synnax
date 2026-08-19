#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Shared base for role permission tests."""

from console.case import ConsoleCase
from x import random_name

PASSWORD = "testpassword123"


class RoleCase(ConsoleCase):
    """Base for role permission tests.

    Subclasses set ``role_name``; ``login_as_role`` registers a fresh user
    with that role and logs in as them.
    """

    role_name: str

    def login_as_role(self) -> str:
        """Register a fresh user with ``role_name`` and log in as them.

        :returns: The generated username.
        """
        username = f"{self.role_name.lower()}_{random_name()}"
        self.log(f"Registering {self.role_name} user: {username}")
        self.console.access.register_user(
            username=username,
            password=PASSWORD,
            first_name=self.role_name,
            last_name="Test",
            role_name=self.role_name,
        )

        self.log(f"Logging out and logging in as {username}")
        self.console.access.logout()
        self.console.access.login(username=username, password=PASSWORD)

        user_badge = self.page.get_by_text(self.role_name, exact=True)
        user_badge.wait_for(state="visible", timeout=10000)
        return username

    def assert_badge_names_role(self) -> None:
        """Assert the user badge reports ``role_name`` for the current session."""
        self.log("Checking the user badge names the role")
        role = self.console.access.get_current_role()
        assert role == self.role_name, (
            f"badge shows role {role!r}, expected {self.role_name!r}"
        )

    def assert_users_toolbar_hidden(self) -> None:
        """Assert the Users toolbar does not open for the current user.

        The toolbar is gated on update permission for users, so only an Owner
        sees it.
        """
        self.log("Checking the Users toolbar stays hidden")
        assert not self.console.access.users_toolbar_visible(), (
            f"Users toolbar is visible to {self.role_name}"
        )

    def assert_command_available(self, command: str) -> None:
        """Assert a command palette entry is offered to the current user."""
        self.log(f"Checking command is available: {command}")
        assert self.console.layout.command_exists(command), (
            f"{command!r} should be available to {self.role_name}"
        )

    def assert_command_hidden(self, command: str) -> None:
        """Assert a command palette entry is hidden from the current user."""
        self.log(f"Checking command is hidden: {command}")
        assert not self.console.layout.command_exists(command), (
            f"{command!r} should be hidden from {self.role_name}"
        )
