#  Copyright 2022 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import pytest

import synnax
from synnax import QueryError, Channel


class TestClient:
    @pytest.fixture(scope="class")
    def two_channels(self, client: synnax.Synnax) -> list[synnax.Channel]:
        return client.channel.create_many(
            [
                Channel(
                    name="test",
                    node_id=1,
                    rate=1 * synnax.HZ,
                    data_type=synnax.FLOAT64,
                ),
                Channel(
                    name="test",
                    node_id=1,
                    rate=1 * synnax.HZ,
                    data_type=synnax.FLOAT64,
                ),
            ]
        )

    def test_create(self, two_channels: list[synnax.Channel]):
        assert len(two_channels) == 2
        for channel in two_channels:
            assert channel.name == "test"
            assert channel.key != ""

    def test_retrieve_by_key(
        self, two_channels: list[synnax.Channel], client: synnax.Synnax
    ) -> None:
        res_channels = client.channel.filter(
            keys=[channel.key for channel in two_channels]
        )
        assert len(res_channels) == 2
        for i, channel in enumerate(res_channels):
            assert two_channels[i].key == channel.key
            assert isinstance(two_channels[i].density, synnax.Density)

    def test_retrieve_by_key_not_found(self, client: synnax.Synnax):
        with pytest.raises(QueryError):
            client.channel.retrieve(key="1-100000")

    def test_retrieve_by_node_id(
        self, two_channels: list[synnax.Channel], client: synnax.Synnax
    ) -> None:
        res_channels = client.channel.filter(node_id=1)
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.node_id == 1

    def test_read_write_data(self, two_channels: list[synnax.Channel]):
        two_channels[0].write(0, np.array([1.0, 2.0, 3.0]))
        data = two_channels[0].read(0, 2 * synnax.SECOND)
        assert np.array_equal(data, np.array([1.0, 2.0]))

    def test_retrieve_by_name(
        self, two_channels: list[synnax.Channel], client: synnax.Synnax
    ) -> None:
        res_channels = client.channel.filter(names=["test"])
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.name == "test"
