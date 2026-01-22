#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test logging out via user badge dropdown."""

from console.case import ConsoleCase
from framework.utils import get_random_name


class UserLogoutBadge(ConsoleCase):
    """Test logging out via the user badge dropdown menu."""

    def run(self) -> None:
        # Create a new user
        username = f"testuser_{get_random_name()}"
        password = "testpassword123"
        first_name = "BadgeTest"
        last_name = "User"
        role_name = "Operator"

        self.log(f"Registering user: {username}")

        success = self.console.access.register_user(
            username=username,
            password=password,
            first_name=first_name,
            last_name=last_name,
            role_name=role_name,
        )
        assert success, f"Failed to register user {username}"

        self.log("User registered, now logging out via badge...")

        # Log out via user badge
        self.console.access.logout_via_badge()
        self.log("Logged out via badge successfully")

        # Log back in as the new user
        self.log(f"Logging in as {username}...")
        self.console.access.login(username, password)

        # Verify we're logged in by checking for the first name in the user badge
        user_badge = self.page.get_by_text(first_name, exact=True)
        user_badge.wait_for(state="visible", timeout=10000)
        self.log(f"Logged in successfully - user badge shows: {first_name}")

        self.log("Logout via badge test passed")
