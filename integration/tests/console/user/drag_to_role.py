#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test changing a user's role via drag-drop in the ontology tree."""

from console.case import ConsoleCase
from framework.utils import get_random_name


class UserDragToRole(ConsoleCase):
    """Test assigning a role to a user via drag-drop."""

    def run(self) -> None:
        # Create a user with initial role
        username = f"testuser_{get_random_name()}"
        password = "testpassword123"
        first_name = "Test"
        last_name = "User"
        initial_role = "Operator"
        new_role = "Viewer"

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

        # Drag user to new role
        self.log(f"Dragging {username} from {initial_role} to {new_role}")
        self.console.access.drag_user_to_role(username, new_role)

        # Expand new role and verify user moved
        self.console.access.expand_role(new_role)
        users_in_new_role = self.console.access.list_users_under_role(new_role)
        self.log(f"Users under {new_role}: {users_in_new_role}")
        assert (
            username in users_in_new_role
        ), f"User {username} not found under {new_role}"

        self.log("User drag-to-role test passed")
