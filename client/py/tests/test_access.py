#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

import pytest

import synnax as sy


@pytest.mark.access
class TestAccessClient:
    @pytest.fixture(scope="class")
    @classmethod
    def role(cls, client: sy.Synnax) -> sy.Role:
        return client.access.roles.create(sy.Role(name="Policy Owner"))

    @pytest.fixture(scope="class")
    @classmethod
    def two_policies(cls, client: sy.Synnax, role: sy.Role) -> list[sy.Policy]:
        return client.access.policies.create(
            [
                sy.Policy(
                    name="Test Policy 1",
                    objects=[
                        sy.resource.ID(type="channel", key=str(uuid.uuid4())),
                        sy.resource.ID(type="label", key=str(uuid.uuid4())),
                    ],
                    actions=["create"],
                    role=role.key,
                ),
                sy.Policy(
                    name="Test Policy 2",
                    objects=[
                        sy.resource.ID(type="channel", key=str(uuid.uuid4())),
                        sy.resource.ID(type="label", key=str(uuid.uuid4())),
                    ],
                    actions=["create"],
                    role=role.key,
                ),
            ]
        )

    def test_create_list(self, two_policies: list[sy.Policy]) -> None:
        assert len(two_policies) == 2
        for policy in two_policies:
            assert "create" in policy.actions
            assert policy.key is not None
            assert policy.name is not None

    def test_create_single(self, client: sy.Synnax, role: sy.Role) -> None:
        p = sy.Policy(
            name="Single Test Policy",
            objects=[
                sy.resource.ID(type="channel", key=str(uuid.uuid4())),
                sy.resource.ID(type="label", key=str(uuid.uuid4())),
            ],
            actions=["create"],
            role=role.key,
        )
        policy = client.access.policies.create(p)
        assert policy.key != ""
        assert policy.actions == ["create"]
        assert policy.name == p.name
        assert policy.objects == p.objects

    def test_create_without_stored_role(self, client: sy.Synnax) -> None:
        """Should reject a policy whose role does not exist."""
        with pytest.raises(sy.ValidationError, match="does not exist"):
            client.access.policies.create(
                sy.Policy(
                    name="Orphan Policy",
                    actions=["retrieve"],
                    role=uuid.uuid4(),
                )
            )

    def test_retrieve_by_role(self, client: sy.Synnax) -> None:
        """Should retrieve the policies of a role."""
        role = client.access.roles.create(sy.Role(name="Policy Holder"))
        policy = client.access.policies.create(
            sy.Policy(name="Role Policy", actions=["retrieve"], role=role.key)
        )
        policies = client.access.policies.retrieve(roles=[role.key])
        assert [p.key for p in policies] == [policy.key]

    def test_retrieve_by_subject(self, client: sy.Synnax) -> None:
        """Should retrieve the policies of the roles a user holds."""
        role = client.access.roles.create(sy.Role(name="Subject Role"))
        policy = client.access.policies.create(
            sy.Policy(name="Subject Policy", actions=["retrieve"], role=role.key)
        )
        user = client.users.create(
            username=str(uuid.uuid4()), password="testpass", roles=[role.key]
        )
        policies = client.access.policies.retrieve(subjects=[user.resource_id])
        assert [p.key for p in policies] == [policy.key]

    def test_retrieve_by_subject_not_found(self, client: sy.Synnax) -> None:
        """Should raise when the subject does not exist."""
        with pytest.raises(sy.NotFoundError):
            client.access.policies.retrieve(
                subjects=[sy.resource.ID(type="user", key=str(uuid.uuid4()))]
            )

    def test_delete_by_key(
        self, two_policies: list[sy.Policy], client: sy.Synnax
    ) -> None:
        key = two_policies[0].key
        assert key is not None
        client.access.policies.delete(key)
        with pytest.raises(sy.NotFoundError):
            client.access.policies.retrieve(keys=[key])

    def test_retrieve_by_internal_flag(self, client: sy.Synnax, role: sy.Role) -> None:
        """Should filter policies by internal flag."""
        # Create a non-internal policy
        created = client.access.policies.create(
            sy.Policy(
                name="Test Non-Internal Policy",
                objects=[sy.resource.ID(type="channel", key=str(uuid.uuid4()))],
                actions=["retrieve"],
                role=role.key,
            )
        )
        assert created.key is not None

        # Retrieve only internal policies (built-in system policies)
        internal_policies = client.access.policies.retrieve(internal=True)
        assert len(internal_policies) > 0
        assert all(p.internal is True for p in internal_policies)
        assert not any(p.key == created.key for p in internal_policies)

        # Retrieve only non-internal policies
        non_internal_policies = client.access.policies.retrieve(internal=False)
        assert all(p.internal is not True for p in non_internal_policies)
        assert any(p.key == created.key for p in non_internal_policies)


@pytest.mark.access
@pytest.mark.role
class TestRoleClient:
    @pytest.fixture(scope="class")
    @classmethod
    def two_roles(cls, client: sy.Synnax) -> list[sy.Role]:
        return client.access.roles.create(
            [
                sy.Role(name="Admin", description="Administrator role"),
                sy.Role(name="Auditor", description="Read-only auditor role"),
            ]
        )

    def test_create_list(self, two_roles: list[sy.Role]) -> None:
        """Should create multiple roles."""
        assert len(two_roles) == 2
        assert two_roles[0].name == "Admin"
        assert two_roles[0].description == "Administrator role"
        assert two_roles[0].key is not None
        assert two_roles[1].name == "Auditor"
        assert two_roles[1].description == "Read-only auditor role"
        assert two_roles[1].key is not None

    def test_create_single(self, client: sy.Synnax) -> None:
        """Should create a single role."""
        role = client.access.roles.create(
            sy.Role(name="Editor", description="Can edit resources")
        )
        assert role.key is not None
        assert role.name == "Editor"
        assert role.description == "Can edit resources"

    def test_retrieve_by_key(self, two_roles: list[sy.Role], client: sy.Synnax) -> None:
        """Should retrieve a role by key."""
        role = client.access.roles.retrieve(key=two_roles[0].key)
        assert role.key == two_roles[0].key
        assert role.name == two_roles[0].name

    def test_retrieve_multiple(
        self, two_roles: list[sy.Role], client: sy.Synnax
    ) -> None:
        """Should retrieve multiple roles."""
        roles = client.access.roles.retrieve(keys=[two_roles[0].key, two_roles[1].key])
        assert len(roles) == 2

    def test_delete(self, client: sy.Synnax) -> None:
        """Should delete a role."""
        role = client.access.roles.create(
            sy.Role(name="Temporary", description="Test role")
        )
        assert role.key is not None
        client.access.roles.delete(role.key)
        # Verify deletion by attempting to retrieve - should raise NotFoundError
        with pytest.raises(sy.NotFoundError):
            client.access.roles.retrieve(keys=[role.key])

    def test_delete_cascades(self, client: sy.Synnax) -> None:
        """Should delete the policies of a role and take it from its users."""
        role = client.access.roles.create(sy.Role(name="Doomed"))
        kept = client.access.roles.create(sy.Role(name="Kept"))
        policy = client.access.policies.create(
            sy.Policy(name="Doomed Policy", actions=["retrieve"], role=role.key)
        )
        user = client.users.create(
            username=str(uuid.uuid4()),
            password="testpass",
            roles=[role.key, kept.key],
        )
        client.access.roles.delete(role.key)
        with pytest.raises(sy.NotFoundError):
            client.access.policies.retrieve(keys=[policy.key])
        assert client.users.retrieve(key=user.key).roles == [kept.key]

    def test_assign_on_create(
        self, two_roles: list[sy.Role], client: sy.Synnax
    ) -> None:
        """Should give a new user the roles it is created with."""
        user = client.users.create(
            username=str(uuid.uuid4()),
            password="testpass",
            roles=[two_roles[0].key],
        )
        assert user.roles == [two_roles[0].key]
        assert client.users.retrieve(key=user.key).roles == [two_roles[0].key]

    def test_set_roles(self, two_roles: list[sy.Role], client: sy.Synnax) -> None:
        """Should remove and add the roles of a user."""
        kept = client.access.roles.create(sy.Role(name="Kept"))
        user = client.users.create(
            username=str(uuid.uuid4()),
            password="testpass",
            roles=[two_roles[0].key, kept.key],
        )
        client.users.set_roles(
            user.key, add=[two_roles[1].key], remove=[two_roles[0].key]
        )
        assert client.users.retrieve(key=user.key).roles == [
            two_roles[1].key,
            kept.key,
        ]

    def test_retrieve_users_by_role(self, client: sy.Synnax) -> None:
        """Should retrieve the users that hold a role."""
        role = client.access.roles.create(sy.Role(name="Holders"))
        holder = client.users.create(
            username=str(uuid.uuid4()), password="testpass", roles=[role.key]
        )
        client.users.create(username=str(uuid.uuid4()), password="testpass")
        users = client.users.retrieve(roles=[role.key])
        assert [u.key for u in users] == [holder.key]

    def test_retrieve_by_internal_flag(self, client: sy.Synnax) -> None:
        """Should filter roles by internal flag."""
        # Create a non-internal role
        created = client.access.roles.create(
            sy.Role(name="Test Non-Internal Role", description="Test role")
        )
        assert created.key is not None

        # Retrieve only internal roles (built-in system roles)
        internal_roles = client.access.roles.retrieve(internal=True)
        assert len(internal_roles) > 0
        assert all(r.internal is True for r in internal_roles)
        assert not any(r.key == created.key for r in internal_roles)

        # Retrieve only non-internal roles
        non_internal_roles = client.access.roles.retrieve(internal=False)
        assert all(r.internal is not True for r in non_internal_roles)
        assert any(r.key == created.key for r in non_internal_roles)


@pytest.mark.access
@pytest.mark.auth
class TestAccessAuthClient:
    def test_new_user_cannot_create_users(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should prevent a newly created user from creating other users."""
        host, port, _, _ = login_info
        username = str(uuid.uuid4())
        client.users.create(username=username, password="pwd2")
        client2 = sy.Synnax(
            host=host,
            port=port,
            username=username,
            password="pwd2",
        )

        with pytest.raises(sy.AccessDenied):
            client2.users.create(username=str(uuid.uuid4()), password="pwd3")
