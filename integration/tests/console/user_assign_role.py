#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test changing a user's role via Console UI context menu."""

import uuid

from console.case import ConsoleCase


class UserAssignRole(ConsoleCase):
    """Test assigning a role to a user via context menu modal."""

    def run(self) -> None:
        # Create a user with initial role
        username = f"testuser_{uuid.uuid4().hex[:8]}"
        password = "testpassword123"
        first_name = "Test"
        last_name = "User"
        initial_role = "Operator"
        new_role = "Engineer"

        self.log(f"Registering user: {username} with role: {initial_role}")

        success = self.console.access.register_user(
            username=username,
            password=password,
            first_name=first_name,
            last_name=last_name,
            role_name=initial_role,
        )
        assert success, f"Failed to register user {username}"

        # Verify user is under initial role
        self.console.access.expand_role(initial_role)
        users = self.console.access.list_users_under_role(initial_role)
        self.log(f"Users under {initial_role}: {users}")
        assert username in users, f"User {username} not found under {initial_role}"

        # Change user's role via context menu
        self.log(f"Changing {username} role from {initial_role} to {new_role}")
        self.console.access.assign_role_to_user(username, new_role)

        # Expand new role and verify user moved
        self.console.access.expand_role(new_role)
        users_in_new_role = self.console.access.list_users_under_role(new_role)
        self.log(f"Users under {new_role}: {users_in_new_role}")
        assert (
            username in users_in_new_role
        ), f"User {username} not found under {new_role}"

        self.log("User role assignment test passed")
