#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test user registration via Console UI."""

import uuid

from console.case import ConsoleCase


class UserRegister(ConsoleCase):
    """Test registering a new user via Console UI."""

    def run(self) -> None:
        username = f"testuser_{uuid.uuid4().hex[:8]}"
        password = "testpassword123"
        first_name = "Test"
        last_name = "User"
        role_name = "Operator"  # Use existing role

        self.log(f"Registering user: {username}")

        # Register via UI
        success = self.console.access.register_user(
            username=username,
            password=password,
            first_name=first_name,
            last_name=last_name,
            role_name=role_name,
        )
        assert success, f"Failed to register user {username}"

        self.log("User registered, verifying in UI...")

        # Debug: list what roles are visible
        roles = self.console.access.list_visible_roles()
        self.log(f"Visible roles: {roles}")

        # Expand the role to see its child users
        self.console.access.expand_role(role_name)

        # Debug: check for any user elements
        user_count = self.page.locator("div[id^='user:']").count()
        self.log(f"User elements found: {user_count}")

        # Verify user appears under the role
        users = self.console.access.list_users_under_role(role_name)
        self.log(f"Users found: {users}")
        assert (
            username in users
        ), f"User {username} not found under {role_name}. Found: {users}"

        self.log("User registration test passed")
