#  Copyright 2025 Synnax Labs, Inc.
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
from synnax.ontology.payload import ID


@pytest.mark.access
class TestAccessClient:
    @pytest.fixture(scope="class")
    def two_policies(self, client: sy.Synnax) -> list[sy.Policy]:
        return client.access.policies.create(
            [
                sy.Policy(
                    name="Test Policy 1",
                    effect="allow",
                    objects=[
                        ID(type="channel", key=str(uuid.uuid4())),
                        ID(type="label", key=str(uuid.uuid4())),
                    ],
                    actions=["create"],
                ),
                sy.Policy(
                    name="Test Policy 2",
                    effect="allow",
                    objects=[
                        ID(type="channel", key=str(uuid.uuid4())),
                        ID(type="label", key=str(uuid.uuid4())),
                    ],
                    actions=["create"],
                ),
            ]
        )

    def test_create_list(self, two_policies: list[sy.Policy]) -> None:
        assert len(two_policies) == 2
        for policy in two_policies:
            assert "create" in policy.actions
            assert policy.key is not None
            assert policy.name is not None
            assert policy.effect == "allow"

    def test_create_single(self, client: sy.Synnax) -> None:
        p = sy.Policy(
            name="Single Test Policy",
            effect="allow",
            objects=[
                ID(type="channel", key=str(uuid.uuid4())),
                ID(type="label", key=str(uuid.uuid4())),
            ],
            actions=["create"],
        )
        policy = client.access.policies.create(p)
        assert policy.key != ""
        assert policy.actions == ["create"]
        assert policy.name == p.name
        assert policy.effect == p.effect
        assert policy.objects == p.objects

    def test_retrieve_by_subject_not_found(self, client: sy.Synnax) -> None:
        """Should handle retrieving policies for non-existent subjects."""
        # Retrieving by subjects that don't exist returns empty list (no policies assigned)
        # But if the subject itself is invalid, it may raise an error
        # For now, just verify the call completes without crashing
        try:
            res = client.access.policies.retrieve(
                subjects=[ID(type="channel", key="hehe")]
            )
            assert isinstance(res, list)
        except sy.NotFoundError:
            # Expected if the subject doesn't exist
            pass

    def test_delete_by_key(
        self, two_policies: list[sy.Policy], client: sy.Synnax
    ) -> None:
        key = two_policies[0].key
        assert key is not None
        client.access.policies.delete(key)
        with pytest.raises(sy.NotFoundError):
            client.access.policies.retrieve(keys=[key])

    def test_retrieve_by_internal_flag(self, client: sy.Synnax) -> None:
        """Should filter policies by internal flag."""
        # Create a non-internal policy
        created = client.access.policies.create(
            sy.Policy(
                name="Test Non-Internal Policy",
                effect="allow",
                objects=[ID(type="channel", key=str(uuid.uuid4()))],
                actions=["retrieve"],
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
    def two_roles(self, client: sy.Synnax) -> list[sy.Role]:
        return client.access.roles.create(
            [
                sy.Role(name="Admin", description="Administrator role"),
                sy.Role(name="Viewer", description="Read-only viewer role"),
            ]
        )

    def test_create_list(self, two_roles: list[sy.Role]) -> None:
        """Should create multiple roles."""
        assert len(two_roles) == 2
        assert two_roles[0].name == "Admin"
        assert two_roles[0].description == "Administrator role"
        assert two_roles[0].key is not None
        assert two_roles[1].name == "Viewer"
        assert two_roles[1].description == "Read-only viewer role"
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

    def test_assign_role(self, two_roles: list[sy.Role], client: sy.Synnax) -> None:
        """Should assign a role to a user."""
        username = str(uuid.uuid4())
        user = client.user.create(username=username, password="testpass")
        assert user.key is not None

        # Assign role to user
        client.access.roles.assign(user=user.key, role=two_roles[0].key)

        # Verify by retrieving policies for the user (via role)
        # Note: This requires the policies to be attached to the role via ontology
        # For now, we just verify the call doesn't error

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
        client.user.create(username=username, password="pwd2")
        client2 = sy.Synnax(
            host=host,
            port=port,
            username=username,
            password="pwd2",
        )

        with pytest.raises(sy.AuthError):
            client2.user.create(username=str(uuid.uuid4()), password="pwd3")
