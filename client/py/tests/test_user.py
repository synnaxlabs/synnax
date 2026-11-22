#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import uuid4 as uuid

import pytest

import synnax as sy
from x import filter
from x.testutil import untyped


@pytest.fixture
def new_user(empty_role: sy.Role) -> sy.user.New:
    return sy.user.New(name="First Last", key=uuid(), role=empty_role.key)


@pytest.fixture
def new_user_list(empty_role: sy.Role) -> list[sy.user.New]:
    role = empty_role.key
    return [
        sy.user.New(name="First1 Last1", role=role),
        sy.user.New(name="First2", key=uuid(), role=role),
    ]


def compare_users(user: sy.user.User, new_user: sy.user.New) -> None:
    assert user.name == new_user.name
    assert user.role == new_user.role
    if new_user.key is not None:
        assert user.key == new_user.key
    else:
        assert user.key is not None


@pytest.mark.user
class TestUserClient:
    def test_create_params(self, client: sy.Synnax, empty_role: sy.Role) -> None:
        role = empty_role.key
        user = client.users.create(name="test", role=role)
        compare_users(user, sy.user.New(name="test", role=role))
        client.users.delete(user.key)
        key = uuid()
        user = client.users.create(name="silly", key=key, role=role)
        compare_users(user, sy.user.New(name="silly", key=key, role=role))
        client.users.delete(user.key)

    def test_create_params_without_name(self, client: sy.Synnax) -> None:
        """Should reject a call with no name, user, or users."""
        create = untyped(client.users.create)
        with pytest.raises(ValueError, match="name, user, or users"):
            create(key=uuid())

    def test_create_params_without_role(self, client: sy.Synnax) -> None:
        """Should reject a name with no role."""
        create = untyped(client.users.create)
        with pytest.raises(ValueError, match="role is required"):
            create(name="test")

    def test_create_user(self, client: sy.Synnax, new_user: sy.user.New) -> None:
        repeated_user = client.users.create(user=new_user)
        compare_users(repeated_user, new_user)

    def test_create_many(
        self, client: sy.Synnax, new_user_list: list[sy.user.New]
    ) -> None:
        repeated_user_list = client.users.create(users=new_user_list)
        assert len(repeated_user_list) == len(new_user_list)
        for i in range(len(repeated_user_list)):
            compare_users(repeated_user_list[i], new_user_list[i])

    def test_change_name(self, client: sy.Synnax, new_user: sy.user.New) -> None:
        user = client.users.create(user=new_user)
        client.users.change_name(user.key, name="Thomas Jefferson")
        retrieved = client.users.retrieve(key=user.key)
        assert retrieved.name == "Thomas Jefferson"

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError for a single user that does not exist."""
        key = uuid()
        with pytest.raises(sy.NotFoundError, match=rf"Found no user where .*{key}"):
            client.users.retrieve(key=key)

    def test_retrieve_missing_keys(
        self, client: sy.Synnax, new_user: sy.user.New
    ) -> None:
        """Should omit keys that do not exist from a list retrieve."""
        user = client.users.create(user=new_user)
        users = client.users.retrieve(keys=[user.key, uuid()])
        assert [u.key for u in users] == [user.key]

    def test_retrieve_empty_keys(self, client: sy.Synnax) -> None:
        """Should return no users for an empty key list."""
        assert client.users.retrieve(keys=[]) == []

    def test_retrieve_where(
        self, client: sy.Synnax, new_user_list: list[sy.user.New]
    ) -> None:
        """Should keep users that match a filter tree and the given keys."""
        users = client.users.create(users=new_user_list)
        where = sy.user.Where(
            not_=sy.user.Where(key=filter.Set[sy.user.Key](eq=users[0].key))
        )
        retrieved = client.users.retrieve(where=where, keys=[u.key for u in users])
        assert [u.key for u in retrieved] == [users[1].key]

    def test_retrieve_one(
        self, client: sy.Synnax, new_user_list: list[sy.user.New]
    ) -> None:
        """Should return one user for any filter when one is set."""
        users = client.users.create(users=new_user_list)
        found = client.users.retrieve(
            keys=[u.key for u in users],
            not_={"key": users[0].key},
            one=True,
        )
        assert found.key == users[1].key

    def test_retrieve_none_roles(self, client: sy.Synnax) -> None:
        """Should reject roles set to None, since it would select every user."""
        with pytest.raises(sy.ValidationError, match="roles is None"):
            untyped(client.users.retrieve)(roles=None)
