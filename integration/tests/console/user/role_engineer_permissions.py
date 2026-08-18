#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Engineer role has full access except user management."""

from tests.console.user.role_case import RoleCase


class RoleEngineerPermissions(RoleCase):
    """Test that Engineer can create resources but cannot manage users."""

    role_name = "Engineer"

    def run(self) -> None:
        self.login_as_role()
        self.assert_users_toolbar_hidden()
        self.assert_command_available("Create project")
        self.assert_command_available("Create schematic")
        self.assert_command_available("Create channel")
