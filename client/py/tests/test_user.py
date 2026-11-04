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
def new_user() -> sy.user.User:
    return sy.user.User(name="First Last", key=uuid())


@pytest.fixture
def new_user_list() -> list[sy.user.User]:
    return [
        sy.user.User(name="First1 Last1"),
        sy.user.User(name="First2", key=uuid()),
    ]


def compare_users(user: sy.user.User, new_user: sy.user.User) -> None:
    assert user.name == new_user.name
    assert user.key == new_user.key


@pytest.mark.user
class TestUserClient:
    def test_create_params(self, client: sy.Synnax) -> None:
        user = client.users.create(name="test")
        assert user.name == "test"
        client.users.delete(user.key)
        key = uuid()
        user = client.users.create(name="silly", key=key)
        compare_users(user, sy.user.User(name="silly", key=key))
        client.users.delete(user.key)

    def test_create_params_without_name(self, client: sy.Synnax) -> None:
        """Should reject a call with no name, user, or users."""
        create = cast(Callable[..., object], client.users.create)
        with pytest.raises(ValueError, match="name, user, or users"):
            create(key=uuid())

    def test_create_user(self, client: sy.Synnax, new_user: sy.user.User) -> None:
        repeated_user = client.users.create(user=new_user)
        compare_users(repeated_user, new_user)

    def test_create_many(
        self, client: sy.Synnax, new_user_list: list[sy.user.User]
    ) -> None:
        repeated_user_list = client.users.create(users=new_user_list)
        assert len(repeated_user_list) == len(new_user_list)
        for i in range(len(repeated_user_list)):
            compare_users(repeated_user_list[i], new_user_list[i])

    def test_change_name(self, client: sy.Synnax, new_user: sy.user.User) -> None:
        user = client.users.create(user=new_user)
        client.users.change_name(user.key, name="Thomas Jefferson")
        retrieved = client.users.retrieve(key=user.key)
        assert retrieved.name == "Thomas Jefferson"
