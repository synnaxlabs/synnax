#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Test that Operator role has restricted permissions."""

from tests.console.user.role_case import RoleCase


class RoleOperatorPermissions(RoleCase):
    """Test that Operator role cannot access user management or create
    resources."""

    role_name = "Operator"

    def run(self) -> None:
        self.login_as_role()
        self.assert_users_toolbar_hidden()
        self.assert_command_hidden("Create project")
        self.assert_command_hidden("Create schematic")
