#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable
from typing import cast
from uuid import uuid4 as uuid

import pytest

import synnax as sy


@pytest.fixture
def new_user() -> sy.user.New:
    return sy.user.New(
        username=uuid().__str__(),
        password="password",
        name="First Last",
        key=uuid(),
    )


@pytest.fixture
def new_user_list() -> list[sy.user.New]:
    return [
        sy.user.New(
            username=uuid().__str__(),
            password="password1",
            name="First1 Last1",
        ),
        sy.user.New(
            username=uuid().__str__(),
            password="password2",
            name="First2",
            key=uuid(),
        ),
        sy.user.New(username=uuid().__str__(), password="password3", key=uuid()),
        sy.user.New(username=uuid().__str__(), password="password4"),
    ]


def compare_users(user: sy.user.User, new_user: sy.user.New) -> None:
    assert user.name == (new_user.name or new_user.username)
    if new_user.key is not None:
        assert user.key == new_user.key
    else:
        assert user.key is not None


@pytest.mark.user
class TestUserClient:
    def test_create_params(self, client: sy.Synnax) -> None:
        user = client.users.create(username="test", password="test")
        compare_users(user, sy.user.New(username="test", password="test"))
        client.users.delete(user.key)
        key = uuid()
        username = uuid().__str__()
        user = client.users.create(
            username=username, password="test", name="silly", key=key
        )
        compare_users(
            user,
            sy.user.New(username=username, password="test", name="silly", key=key),
        )
        client.users.delete(user.key)

    def test_create_params_without_password(self, client: sy.Synnax) -> None:
        """Should reject a username with no password."""
        create = cast(Callable[..., object], client.users.create)
        with pytest.raises(ValueError, match="password is required"):
            create(username=str(uuid()))

    def test_create_user(self, client: sy.Synnax, new_user: sy.user.New) -> None:
        repeated_user = client.users.create(user=new_user)
        compare_users(repeated_user, new_user)

    def test_create_repeated(self, client: sy.Synnax) -> None:
        with pytest.raises(sy.AuthError):
            client.users.create(username="synnax", password="test")

    def test_create_many(
        self, client: sy.Synnax, new_user_list: list[sy.user.New]
    ) -> None:
        repeated_user_list = client.users.create(users=new_user_list)
        assert len(repeated_user_list) == len(new_user_list)
        for i in range(len(repeated_user_list)):
            compare_users(repeated_user_list[i], new_user_list[i])

    def test_change_username(self, client: sy.Synnax, new_user: sy.user.New) -> None:
        user = client.users.create(user=new_user)
        username = str(uuid())
        client.users.change_username(user.key, username)
        assert client.users.retrieve(username=username).key == user.key

    def test_change_name(self, client: sy.Synnax, new_user: sy.user.New) -> None:
        user = client.users.create(user=new_user)
        client.users.change_name(user.key, name="Thomas Jefferson")
        retrieved = client.users.retrieve(key=user.key)
        assert retrieved.name == "Thomas Jefferson"

    def test_retrieve_by_username(
        self, client: sy.Synnax, new_user: sy.user.New
    ) -> None:
        """Should retrieve one user by username."""
        user = client.users.create(user=new_user)
        assert client.users.retrieve(username=new_user.username).key == user.key

    def test_retrieve_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError for a single user that does not exist."""
        key = uuid()
        with pytest.raises(sy.NotFoundError, match=f"User matching {key} not found"):
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

    def test_retrieve_conditions(
        self, client: sy.Synnax, new_user_list: list[sy.user.New]
    ) -> None:
        """Should keep users that match every condition and the given usernames."""
        new_users = new_user_list[:2]
        users = client.users.create(users=new_users)
        retrieved = client.users.retrieve(
            ~sy.user.key.eq(users[0].key), usernames=[u.username for u in new_users]
        )
        assert [u.key for u in retrieved] == [users[1].key]

    def test_retrieve_by_bare_username(
        self, client: sy.Synnax, new_user_list: list[sy.user.New]
    ) -> None:
        """Should retrieve a user by a key or a username as the first argument."""
        new_users = new_user_list[:2]
        users = client.users.create(users=new_users)
        assert client.users.retrieve(new_users[0].username).key == users[0].key
        assert client.users.retrieve(users[0].key).name == users[0].name
        usernames = [u.username for u in new_users]
        assert len(client.users.retrieve(usernames)) == 2
        with pytest.raises(sy.MultipleFoundError):
            client.users.retrieve(sy.user.usernames.has_any(usernames), one=True)
