#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Shared base for role permission tests."""

import synnax as sy
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

    def assert_users_toolbar_hidden(self) -> None:
        """Assert the Users toolbar does not open for the current user.

        The toolbar is gated on update permission for users; its "u" hotkey is
        suppressed when hidden, so pressing it must not reveal role items.
        """
        self.log("Checking the Users toolbar stays hidden")
        self.page.keyboard.press("u")
        sy.sleep(0.5)
        role_elements = self.page.locator("div[id^='role:']")
        visible = role_elements.count() > 0 and role_elements.first.is_visible()
        assert not visible, f"Users toolbar is visible to {self.role_name}"
        self.console.layout.press_escape()

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
