#  Copyright 2023 Synnax Labs, Inc.
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
from synnax.channel.client import channel_ontology_type
from synnax.ontology.id import OntologyID


@pytest.mark.access
class TestAccessClient:
    @pytest.fixture(scope="class")
    def two_policies(self, client: sy.Synnax) -> list[sy.Policy]:
        return client.access.create(
            [
                sy.Policy(
                    subjects=[OntologyID(type="user", key=str(uuid.uuid4()))],
                    objects=[
                        OntologyID(type="channel", key=str(uuid.uuid4())),
                        OntologyID(type="label", key=str(uuid.uuid4())),
                    ],
                    actions=["create"],
                ),
                sy.Policy(
                    subjects=[OntologyID(type="user", key=str(uuid.uuid4()))],
                    objects=[
                        OntologyID(type="channel", key=str(uuid.uuid4())),
                        OntologyID(type="label", key=str(uuid.uuid4())),
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
            subjects=[OntologyID(type="user", key=str(uuid.uuid4()))],
            objects=[
                OntologyID(type="channel", key=str(uuid.uuid4())),
                OntologyID(type="label", key=str(uuid.uuid4())),
            ],
            actions=["create"],
        )
        policy = client.access.create(p)
        assert policy.key != ""
        assert policy.actions == ["create"]
        assert policy.subjects == p.subjects
        assert policy.objects == p.objects

    def test_create_from_kwargs(self, client: sy.Synnax):
        resourceID = str(uuid.uuid4())
        policy = client.access.create(
            subjects=[OntologyID(type="user", key=resourceID)],
            objects=[
                OntologyID(type="channel", key=resourceID),
                OntologyID(type="label", key=resourceID),
            ],
            actions=["create"],
        )
        assert policy.key != ""
        assert policy.actions == ["create"]
        assert policy.subjects == [OntologyID(type="user", key=resourceID)]
        assert policy.objects == [
            OntologyID(type="channel", key=resourceID),
            OntologyID(type="label", key=resourceID),
        ]

    def test_retrieve_by_subject(
        self, two_policies: list[sy.Policy], client: sy.Synnax
    ) -> None:
        p = client.access.retrieve(two_policies[0].subjects[0])
        assert p.actions == ["create"]
        assert (p.objects[0].type, p.objects[1].type) == ("channel", "label")

    def test_retrieve_by_subject_not_found(self, client: sy.Synnax):
        with pytest.raises(sy.NotFoundError):
            client.access.retrieve(OntologyID(type="channel", key="hehe"))

    def test_delete_by_key(self, two_policies: list[sy.Policy], client: sy.Synnax):
        client.access.delete(two_policies[0].key)
        with pytest.raises(sy.QueryError):
            client.access.retrieve(two_policies[0].subjects[0])


@pytest.mark.access
@pytest.mark.auth
class TestAccessAuthClient:
    def test_create_user(self, client: sy.Synnax):
        username = str(uuid.uuid4())
        client.user.register(username, "pwd2")
        sy.Synnax(
            host="localhost",
            port=9090,
            username=username,
            password="pwd2",
        )

    def test_user_privileges(self, client: sy.Synnax):
        username = str(uuid.uuid4())
        usr = client.user.register(username, "pwd3")
        client2 = sy.Synnax(
            host="localhost",
            port=9090,
            username=username,
            password="pwd3",
        )

        with pytest.raises(sy.AuthError):
            client2.channels.create(
                sy.Channel(
                    name="new_channel",
                    data_type=sy.DataType.FLOAT32,
                    rate=1 * sy.Rate.HZ,
                )
            )

        p = client.access.create(
            subjects=[usr.ontology_id()],
            objects=[channel_ontology_type],
            actions=["create"],
        )

        client2.channels.create(
            sy.Channel(
                name="new_channel",
                data_type=sy.DataType.FLOAT32,
                rate=1 * sy.Rate.HZ,
            )
        )

        # revoke the policy
        client.access.delete(p.key)

        with pytest.raises(sy.AuthError):
            client2.channels.create(
                sy.Channel(
                    name="new_channel",
                    data_type=sy.DataType.FLOAT32,
                    rate=1 * sy.Rate.HZ,
                )
            )
