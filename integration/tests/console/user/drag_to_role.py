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
from x import random_name


class UserDragToRole(ConsoleCase):
    """Test assigning a role to a user via drag-drop."""

    def run(self) -> None:
        # Create a user with initial role
        username = f"testuser_{random_name()}"
        password = "testpassword123"
        initial_role = "Operator"
        new_role = "Viewer"

        self.log(f"Registering user: {username} with role: {initial_role}")

        self.console.access.register_user(
            username=username,
            password=password,
            role_name=initial_role,
        )

        # Verify user is under initial role
        self.console.access.expand_role(initial_role)
        self.console.access.wait_for_user_role(username, role_name=initial_role)

        # Drag user to new role
        self.log(f"Dragging {username} from {initial_role} to {new_role}")
        self.console.access.drag_user_to_role(username, new_role)

        # Expand new role and verify user moved
        self.console.access.expand_role(new_role)
        self.console.access.wait_for_user_role(
            username, role_name=new_role, previous_role=initial_role
        )

        self.log("User drag-to-role test passed")
