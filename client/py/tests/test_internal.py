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


@pytest.mark.internal
class TestInternalClient:
    @pytest.fixture(scope="class")
    def hundred_channels(self, client: sy.Synnax) -> list[sy.Channel]:
        listicle = []
        for i in range(100):
            listicle.append(
                sy.Channel(
                    name=f"sensor_{i+1}",
                    rate=1 * sy.Rate.HZ,
                    data_type=sy.DataType.FLOAT64,
                    internal=True,
                )
            )
        return client.channels.create(listicle)

    def test_create_list(self, hundred_channels: list[sy.Channel]):
        """Should create a list of 100 valid channels"""
        assert len(hundred_channels) == 100
        for channel in hundred_channels:
            assert channel.name.startswith("sensor")
            assert channel.key != ""

    def test_retrieve_by_key(
        self, hundred_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        """Should retrieve one hundred channels using a list of keys"""
        res_channels = client.channels.retrieve(
            [channel.key for channel in hundred_channels]
        )
        assert len(res_channels) == 100
        for i, ch in enumerate(res_channels):
            assert hundred_channels[i].key == ch.key
