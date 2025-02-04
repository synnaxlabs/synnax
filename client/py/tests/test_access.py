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
from synnax.user.payload import ontology_type as USER_ONTOLOGY_TYPE
from synnax.ontology.payload import ID


@pytest.mark.access
class TestAccessClient:
    @pytest.fixture(scope="class")
    def two_policies(self, client: sy.Synnax) -> list[sy.Policy]:
        return client.access.create(
            [
                sy.Policy(
                    subjects=[ID(type="user", key=str(uuid.uuid4()))],
                    objects=[
                        ID(type="channel", key=str(uuid.uuid4())),
                        ID(type="label", key=str(uuid.uuid4())),
                    ],
                    actions=["create"],
                ),
                sy.Policy(
                    subjects=[ID(type="user", key=str(uuid.uuid4()))],
                    objects=[
                        ID(type="channel", key=str(uuid.uuid4())),
                        ID(type="label", key=str(uuid.uuid4())),
                    ],
                    actions=["create"],
                ),
            ]
        )

    def test_create_list(self, two_policies: list[sy.Policy]):
        assert len(two_policies) == 2
        for policy in two_policies:
            assert "create" in policy.actions
            assert policy.key is not None

    def test_create_single(self, client: sy.Synnax):
        p = sy.Policy(
            subjects=[ID(type="user", key=str(uuid.uuid4()))],
            objects=[
                ID(type="channel", key=str(uuid.uuid4())),
                ID(type="label", key=str(uuid.uuid4())),
            ],
            actions=["create"],
        )
        policy = client.access.create(p)
        assert policy.key != ""
        assert policy.actions == ["create"]
        assert policy.subjects == p.subjects
        assert policy.objects == p.objects

    def test_create_from_kwargs(self, client: sy.Synnax):
        resource_id = str(uuid.uuid4())
        policy = client.access.create(
            subjects=[ID(type="user", key=resource_id)],
            objects=[
                ID(type="channel", key=resource_id),
                ID(type="label", key=resource_id),
            ],
            actions=["create"],
        )
        assert policy.key != ""
        assert policy.actions == ["create"]
        assert policy.subjects == [ID(type="user", key=resource_id)]
        assert policy.objects == [
            ID(type="channel", key=resource_id),
            ID(type="label", key=resource_id),
        ]

    def test_retrieve_by_subject_not_found(self, client: sy.Synnax):
        res = client.access.retrieve(subjects=[ID(type="channel", key="hehe")])
        assert res is None

    def test_delete_by_key(self, two_policies: list[sy.Policy], client: sy.Synnax):
        client.access.delete(two_policies[0].key)
        with pytest.raises(sy.NotFoundError):
            client.access.retrieve(keys=[two_policies[0].key])


@pytest.mark.access
@pytest.mark.auth
class TestAccessAuthClient:
    def test_create_user(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ):
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

    def test_user_privileges(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ):
        host, port, _, _ = login_info
        username = str(uuid.uuid4())
        usr = client.user.create(username=username, password="pwd3")
        client2 = sy.Synnax(
            host=host,
            port=port,
            username=username,
            password="pwd3",
        )

        with pytest.raises(sy.AuthError):
            client2.user.create(username=str(uuid.uuid4()), password="pwd3")
