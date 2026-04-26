#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration verify: confirm RBAC state survived migration."""

from uuid import UUID

import synnax as sy
from console.case import ConsoleCase
from tests.migration.rbac_setup import (
    CUSTOM_ROLE_DESCRIPTION,
    CUSTOM_ROLE_NAME,
    PASSWORD,
    USERS_SPEC,
)

BUILTIN_ROLES = ["Owner", "Engineer", "Operator", "Viewer"]


def _get_user_role_names(client: sy.Synnax, username: str) -> list[str]:
    user = client.users.retrieve(username=username)
    parents = client.ontology.retrieve_parents(user.ontology_id)
    return [r.name for r in parents if r.id.type == "role"]


class RBACVerify(ConsoleCase):
    """Verify RBAC state survived migration via API and console UI."""

    custom_role_key: UUID

    def run(self) -> None:
        self.test_custom_role()
        self.test_users()
        self.test_role_assignments()
        self.test_builtin_roles()
        self.test_users_in_console()
        self.test_user_logins()

    def test_custom_role(self) -> None:
        self.log("Testing: Custom role survived migration")
        roles = self.client.access.roles.retrieve(internal=False)
        match = [r for r in roles if r.name == CUSTOM_ROLE_NAME]
        assert len(match) >= 1, (
            f"Custom role '{CUSTOM_ROLE_NAME}' not found in {[r.name for r in roles]}"
        )
        role = match[0]
        assert role.description == CUSTOM_ROLE_DESCRIPTION, (
            f"Description mismatch: '{role.description}' != '{CUSTOM_ROLE_DESCRIPTION}'"
        )
        assert role.key is not None
        self.custom_role_key = role.key

    def test_users(self) -> None:
        self.log("Testing: Users survived migration")
        for username, first_name, last_name, _ in USERS_SPEC:
            user = self.client.users.retrieve(username=username)
            assert user.first_name == first_name, (
                f"User '{username}' first name: '{user.first_name}' != '{first_name}'"
            )
            assert user.last_name == last_name, (
                f"User '{username}' last name: '{user.last_name}' != '{last_name}'"
            )

    def test_role_assignments(self) -> None:
        self.log("Testing: Role assignments survived migration")
        for username, _, _, role in USERS_SPEC:
            role_names = _get_user_role_names(self.client, username)
            assert role in role_names, (
                f"User '{username}': expected role '{role}', got {role_names}"
            )

    def test_builtin_roles(self) -> None:
        self.log("Testing: Built-in roles survived migration")
        internal = self.client.access.roles.retrieve(internal=True)
        names = {r.name for r in internal}
        for expected in BUILTIN_ROLES:
            assert expected in names, (
                f"Built-in role '{expected}' not found. Got: {names}"
            )

    def test_users_in_console(self) -> None:
        self.log("Testing: Users visible under correct roles in console")
        visible_roles = self.console.access.list_visible_roles()
        assert CUSTOM_ROLE_NAME in visible_roles, (
            f"Role '{CUSTOM_ROLE_NAME}' not in console: {visible_roles}"
        )
        for username, _, _, role in USERS_SPEC:
            self.console.access.expand_role(role)
            users = self.console.access.list_users_under_role(role)
            assert username in users, (
                f"User '{username}' not under role '{role}': {users}"
            )

    def test_user_logins(self) -> None:
        self.log("Testing: Each user can log in after migration")
        for username, first_name, _, _ in USERS_SPEC:
            self.console.access.logout()
            self.console.access.login(username=username, password=PASSWORD)
            badge = self.page.get_by_text(first_name, exact=True)
            badge.wait_for(state="visible", timeout=10000)
