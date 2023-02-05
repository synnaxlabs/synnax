#  Copyright 2023 sy Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest

import synnax as sy


class TestClient:
    @pytest.fixture(scope="class")
    def two_channels(self, client: sy.Synnax) -> list[sy.Channel]:
        return client.channels.create(
            [
                sy.Channel(
                    name="test",
                    node_id=1,
                    rate=1 * sy.Rate.HZ,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name="test2",
                    node_id=1,
                    rate=1 * sy.Rate.HZ,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )

    def test_create(self, two_channels: list[sy.Channel]):
        assert len(two_channels) == 2
        for channel in two_channels:
            assert channel.name.startswith("test")
            assert channel.key != ""

    def test_retrieve_by_key(
        self, two_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        res_channels = client.channels.retrieve(
            keys=[channel.key for channel in two_channels]
        )
        assert len(res_channels) == 2
        for i, channel in enumerate(res_channels):
            assert two_channels[i].key == channel.key
            assert isinstance(two_channels[i].density, sy.Density)

    def test_retrieve_by_key_not_found(self, client: sy.Synnax):
        with pytest.raises(sy.QueryError):
            client.channels.retrieve(keys="1-100000")

    def test_retrieve_by_node_id(
        self, two_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        res_channels = client.channels.retrieve(node_id=1)
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.node_id == 1

    def test_retrieve_by_name(
        self, two_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        res_channels = client.channels.retrieve(names=["test", "test2"])
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.name in ["test", "test2"]
