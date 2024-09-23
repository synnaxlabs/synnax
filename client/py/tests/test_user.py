#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
import synnax as sy
from uuid import uuid4 as uuid
from synnax.user.payload import NewUser, User


def pytest_configure():
    new_user = NewUser(
        username="username",
        password="password",
        first_name="first",
        last_name="last",
        key=uuid(),
    )
    repeated_user: User

    new_user_list = [
        NewUser(
            username="username1",
            password="password1",
            first_name="first1",
            last_name="last1",
        ),
        NewUser(
            username="username2",
            password="password2",
            first_name="first2",
            key=uuid(),
        ),
        NewUser(username="username3", password="password3", key=uuid()),
        NewUser(username="username4", password="password4"),
    ]

    repeated_user_list: list[User]


def compare_users(user: User, new_user: NewUser):
    assert user.username == new_user.username
    assert user.first_name == new_user.first_name
    assert user.last_name == new_user.last_name
    if new_user.key is not None:
        assert user.key == new_user.key
    else:
        assert user.key is not None


@pytest.fixture
class TestUserClient:

    def test_create_params(self, client: sy.Synnax):
        user = client.user.create(username="test", password="test")
        compare_users(user, NewUser(username="test", password="test"))
        client.user.delete(user.key)
        key = uuid()
        user = client.user.create(
            username="new_username", password="test", first_name="silly", key=key
        )
        compare_users(
            user,
            NewUser(
                username="new_username", password="test", first_name="silly", key=key
            ),
        )
        client.user.delete(user.key)

    def test_create_user(self, client: sy.Synnax):
        repeated_user = client.user.create(user=new_user)
        compare_users(repeated_user, new_user)

    def test_create_repeated(self, client: sy.Synnax):
        with pytest.raises(sy.AuthError):
            client.user.create(username="username", password="test")

    def test_create_many(self, client: sy.Synnax):
        repeated_user_list = client.user.create(users=new_user_list)
        assert len(repeated_user_list) == len(new_user_list)
        for i in range(len(repeated_user_list)):
            compare_users(repeated_user_list[i], new_user_list[i])
