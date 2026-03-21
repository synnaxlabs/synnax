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


@pytest.fixture
def new_user():
    return sy.user.New(
        username=uuid().__str__(),
        password="password",
        first_name="first",
        last_name="last",
        key=uuid(),
    )


@pytest.fixture
def new_user_list():
    return [
        sy.user.New(
            username=uuid().__str__(),
            password="password1",
            first_name="first1",
            last_name="last1",
        ),
        sy.user.New(
            username=uuid().__str__(),
            password="password2",
            first_name="first2",
            key=uuid(),
        ),
        sy.user.New(username=uuid().__str__(), password="password3", key=uuid()),
        sy.user.New(username=uuid().__str__(), password="password4"),
    ]


def compare_users(user: sy.user.User, new_user: sy.user.New):
    assert user.username == new_user.username
    assert user.first_name == new_user.first_name
    assert user.last_name == new_user.last_name
    if new_user.key is not None:
        assert user.key == new_user.key
    else:
        assert user.key is not None


@pytest.mark.user
class TestUserClient:
    def test_create_params(self, client: sy.Synnax):
        user = client.users.create(username="test", password="test")
        compare_users(user, sy.user.New(username="test", password="test"))
        client.users.delete(user.key)
        key = uuid()
        username = uuid().__str__()
        user = client.users.create(
            username=username, password="test", first_name="silly", key=key
        )
        compare_users(
            user,
            sy.user.New(
                username=username, password="test", first_name="silly", key=key
            ),
        )
        client.users.delete(user.key)

    def test_create_user(self, client: sy.Synnax, new_user):
        repeated_user = client.users.create(user=new_user)
        compare_users(repeated_user, new_user)

    def test_create_repeated(self, client: sy.Synnax):
        with pytest.raises(sy.AuthError):
            client.users.create(username="synnax", password="test")

    def test_create_many(self, client: sy.Synnax, new_user_list):
        repeated_user_list = client.users.create(users=new_user_list)
        assert len(repeated_user_list) == len(new_user_list)
        for i in range(len(repeated_user_list)):
            compare_users(repeated_user_list[i], new_user_list[i])
