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
from synnax.channel.client import CHANNEL_ONTOLOGY_TYPE
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

    def test_retrieve_by_subject(
        self, two_policies: list[sy.Policy], client: sy.Synnax
    ) -> None:
        p = client.access.retrieve(two_policies[0].subjects[0])
        assert len(p) == 1
        assert p[0].actions == ["create"]
        assert (p[0].objects[0].type, p[0].objects[1].type) == ("channel", "label")

    def test_retrieve_by_subject_not_found(self, client: sy.Synnax):
        res = client.access.retrieve(ID(type="channel", key="hehe"))
        assert len(res) == 0

    def test_delete_by_key(self, two_policies: list[sy.Policy], client: sy.Synnax):
        client.access.delete(two_policies[0].key)
        p = client.access.retrieve(two_policies[0].subjects[0])
        assert len(p) == 0


@pytest.mark.access
@pytest.mark.auth
class TestAccessAuthClient:
    def test_create_user(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ):
        host, port, _, _ = login_info
        username = str(uuid.uuid4())
        client.user.register(username, "pwd2")
        client2 = sy.Synnax(
            host=host,
            port=port,
            username=username,
            password="pwd2",
        )

        with pytest.raises(sy.AuthError):
            client2.user.register(str(uuid.uuid4()), "pwd3")

    def test_user_privileges(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ):
        host, port, _, _ = login_info
        username = str(uuid.uuid4())
        usr = client.user.register(username, "pwd3")
        client2 = sy.Synnax(
            host=host,
            port=port,
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
            objects=[CHANNEL_ONTOLOGY_TYPE],
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

    def test_privilege_framer(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ):
        host, port, _, _ = login_info
        username = str(uuid.uuid4())
        usr = client.user.register(username, "pwd3")
        client2 = sy.Synnax(
            host=host,
            port=port,
            username=username,
            password="pwd3",
        )

        client.access.create(
            subjects=[usr.ontology_id()],
            objects=[CHANNEL_ONTOLOGY_TYPE],
            actions=["create"],
        )

        chs = client2.channels.create(
            [
                sy.Channel(
                    name="new_channel",
                    data_type=sy.DataType.FLOAT32,
                    rate=1 * sy.Rate.HZ,
                ),
                sy.Channel(
                    name="new_channel_with_perm",
                    data_type=sy.DataType.FLOAT32,
                    rate=1 * sy.Rate.HZ,
                ),
            ]
        )

        client.access.create(
            subjects=[usr.ontology_id()],
            objects=[sy.ontology.OntologyID(key=chs[1].key, type="framer")],
            actions=["retrieve"],
        )

        with pytest.raises(sy.AuthError):
            client2.open_iterator(sy.TimeRange.MAX, [ch.key for ch in chs])

        with pytest.raises(sy.AuthError):
            client2.open_writer(0, [ch.key for ch in chs])

        with pytest.raises(sy.AuthError):
            client2.open_streamer([ch.key for ch in chs])

        # Assert that opening on channels that have allowed policy is fine
        i = client2.open_iterator(sy.TimeRange.MAX, chs[1].key)
        s = client2.open_streamer(chs[1].key)
        with pytest.raises(sy.AuthError):
            client2.open_writer(0, chs[1].key)

        i.close()
        s.close()

    def test_user_privileges_allow_all(
        self, client: sy.Synnax, login_info: tuple[str, int, str, str]
    ):
        host, port, _, _ = login_info
        username = str(uuid.uuid4())
        usr = client.user.register(username, "pwd4")
        client2 = sy.Synnax(
            host=host,
            port=port,
            username=username,
            password="pwd4",
        )

        p = client.access.create(
            subjects=[usr.ontology_id()],
            objects=[sy.access.ALLOW_ALL],
            actions=[],
        )

        client2.channels.create(
            sy.Channel(
                name="new_channel",
                data_type=sy.DataType.FLOAT32,
                rate=1 * sy.Rate.HZ,
            )
        )

        client2.ranges.create(
            name="range1", time_range=sy.TimeStamp(1).span_range(2 * sy.TimeSpan.SECOND)
        )
