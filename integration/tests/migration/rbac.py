#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Migration test: create RBAC state on old version, verify after migration."""

from abc import abstractmethod
from dataclasses import dataclass
from uuid import UUID

import synnax as sy

from console.case import ConsoleCase
from framework.test_case import TestCase

CUSTOM_ROLE_NAME = "mig_rbac_role"
CUSTOM_ROLE_DESCRIPTION = "Custom role for migration testing"

PASSWORD = "mig_rbac_pass123"

BUILTIN_ROLES = ["Owner", "Engineer", "Operator", "Viewer"]


@dataclass
class UserSpec:
    username: str
    first_name: str
    last_name: str
    role: str


USERS: list[UserSpec] = [
    UserSpec(
        username="mig_rbac_custom",
        first_name="MigCustom",
        last_name="RbacUser",
        role=CUSTOM_ROLE_NAME,
    ),
    UserSpec(
        username="mig_rbac_operator",
        first_name="MigOperator",
        last_name="RbacUser",
        role="Operator",
    ),
    UserSpec(
        username="mig_rbac_viewer",
        first_name="MigViewer",
        last_name="RbacUser",
        role="Viewer",
    ),
]


def _get_user_role_names(client: sy.Synnax, username: str) -> list[str]:
    user = client.users.retrieve(username=username)
    parents = client.ontology.retrieve_parents(user.ontology_id)
    return [r.name for r in parents if r.id.type == "role"]


class RBACMigration(TestCase):
    """Base class defining the migration test contract for RBAC.

    Subclasses must implement each test method — setup creates the state,
    verify checks it after migration.
    """

    custom_role_key: UUID

    def run(self) -> None:
        self.test_custom_role()
        self.test_users()
        self.test_role_assignments()
        self.test_builtin_roles()

    @abstractmethod
    def test_custom_role(self) -> None: ...

    @abstractmethod
    def test_users(self) -> None: ...

    @abstractmethod
    def test_role_assignments(self) -> None: ...

    @abstractmethod
    def test_builtin_roles(self) -> None: ...


class RBACSetup(RBACMigration):
    """Create custom role, users across roles, and verify built-in roles."""

    def test_custom_role(self) -> None:
        self.log("Testing: Create custom role")
        role = self.client.access.roles.create(
            sy.Role(name=CUSTOM_ROLE_NAME, description=CUSTOM_ROLE_DESCRIPTION)
        )
        assert role.key is not None
        assert role.name == CUSTOM_ROLE_NAME
        assert role.description == CUSTOM_ROLE_DESCRIPTION
        self.custom_role_key = role.key

    def test_users(self) -> None:
        self.log("Testing: Create users")
        for spec in USERS:
            user = self.client.users.create(
                username=spec.username,
                password=PASSWORD,
                first_name=spec.first_name,
                last_name=spec.last_name,
            )
            assert user.key is not None
            assert user.username == spec.username

    def test_role_assignments(self) -> None:
        self.log("Testing: Assign roles to users")
        internal = self.client.access.roles.retrieve(internal=True)
        builtin_by_name = {r.name: r for r in internal}

        for spec in USERS:
            user = self.client.users.retrieve(username=spec.username)
            if spec.role == CUSTOM_ROLE_NAME:
                role_key = self.custom_role_key
            else:
                role_key = builtin_by_name[spec.role].key
            assert role_key is not None
            self.client.access.roles.assign(user=user.key, role=role_key)

            role_names = _get_user_role_names(self.client, spec.username)
            assert spec.role in role_names, (
                f"User '{spec.username}': expected role '{spec.role}', "
                f"got {role_names}"
            )

    def test_builtin_roles(self) -> None:
        self.log("Testing: Built-in roles exist")
        internal = self.client.access.roles.retrieve(internal=True)
        names = {r.name for r in internal}
        for expected in BUILTIN_ROLES:
            assert (
                expected in names
            ), f"Built-in role '{expected}' not found. Got: {names}"


class RBACVerify(ConsoleCase, RBACMigration):
    """Verify RBAC state survived migration via API and console UI."""

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
            f"Custom role '{CUSTOM_ROLE_NAME}' not found in "
            f"{[r.name for r in roles]}"
        )
        role = match[0]
        assert role.description == CUSTOM_ROLE_DESCRIPTION, (
            f"Description mismatch: '{role.description}' != "
            f"'{CUSTOM_ROLE_DESCRIPTION}'"
        )
        assert role.key is not None
        self.custom_role_key = role.key

    def test_users(self) -> None:
        self.log("Testing: Users survived migration")
        for spec in USERS:
            user = self.client.users.retrieve(username=spec.username)
            assert user.first_name == spec.first_name, (
                f"User '{spec.username}' first name: "
                f"'{user.first_name}' != '{spec.first_name}'"
            )
            assert user.last_name == spec.last_name, (
                f"User '{spec.username}' last name: "
                f"'{user.last_name}' != '{spec.last_name}'"
            )

    def test_role_assignments(self) -> None:
        self.log("Testing: Role assignments survived migration")
        for spec in USERS:
            role_names = _get_user_role_names(self.client, spec.username)
            assert spec.role in role_names, (
                f"User '{spec.username}': expected role '{spec.role}', "
                f"got {role_names}"
            )

    def test_builtin_roles(self) -> None:
        self.log("Testing: Built-in roles survived migration")
        internal = self.client.access.roles.retrieve(internal=True)
        names = {r.name for r in internal}
        for expected in BUILTIN_ROLES:
            assert (
                expected in names
            ), f"Built-in role '{expected}' not found. Got: {names}"

    def test_users_in_console(self) -> None:
        self.log("Testing: Users visible under correct roles in console")
        visible_roles = self.console.access.list_visible_roles()
        assert (
            CUSTOM_ROLE_NAME in visible_roles
        ), f"Role '{CUSTOM_ROLE_NAME}' not in console: {visible_roles}"
        for spec in USERS:
            self.console.access.expand_role(spec.role)
            users = self.console.access.list_users_under_role(spec.role)
            assert spec.username in users, (
                f"User '{spec.username}' not under role " f"'{spec.role}': {users}"
            )

    def test_user_logins(self) -> None:
        self.log("Testing: Each user can log in after migration")
        for spec in USERS:
            self.console.access.logout()
            self.console.access.login(username=spec.username, password=PASSWORD)
            badge = self.page.get_by_text(spec.first_name, exact=True)
            badge.wait_for(state="visible", timeout=10000)
