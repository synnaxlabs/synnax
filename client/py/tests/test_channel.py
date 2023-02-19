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
import numpy as np


class TestChannelClient:
    @pytest.fixture(scope="class")
    @pytest.mark.channel
    def two_channels(self, client: sy.Synnax) -> list[sy.Channel]:
        return client.channels.create(
            [
                sy.Channel(
                    name="test",
                    rate=1 * sy.Rate.HZ,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name="test2",
                    rate=1 * sy.Rate.HZ,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )

    @pytest.mark.channel
    def test_write_read(self, client: sy.Synnax):
        """Should create a channel and write then read from it"""
        channel = client.channels.create(
            sy.Channel(
                name="test",
                rate=1 * sy.Rate.HZ,
                data_type=sy.DataType.FLOAT64,
            )
        )
        input = np.array([1, 2, 3, 4, 5])
        start = 1 * sy.TimeSpan.SECOND
        channel.write(start, input)
        data, tr = channel.read(start, (start + input) * sy.TimeSpan.SECOND)
        assert len(input) == len(data)
        assert input[0] == data[0]
        assert tr.start == start
        assert tr.end == (start + len(input) - 1) * sy.TimeSpan.SECOND + 1

    @pytest.mark.channel
    def test_create_list(self, two_channels: list[sy.Channel]):
        """Should create a list of valid channels"""
        assert len(two_channels) == 2
        for channel in two_channels:
            assert channel.name.startswith("test")
            assert channel.key != ""

    @pytest.mark.channel
    def test_create_single(self, client: sy.Synnax):
        """Should create a single valid channel"""
        channel = client.channels.create(
            sy.Channel(
                name="test",
                rate=1 * sy.Rate.HZ,
                data_type=sy.DataType.FLOAT64,
            )
        )
        assert channel.name == "test"
        assert channel.key != ""
        assert channel.data_type == sy.DataType.FLOAT64
        assert channel.rate == 1 * sy.Rate.HZ

    @pytest.mark.channel
    def test_create_from_kwargs(self, client: sy.Synnax):
        """Should create a single valid channel"""
        channel = client.channels.create(
            name="test",
            rate=1 * sy.Rate.HZ,
            data_type=sy.DataType.FLOAT64,
        )
        assert channel.name == "test"
        assert channel.key != ""
        assert channel.data_type == sy.DataType.FLOAT64
        assert channel.rate == 1 * sy.Rate.HZ

    @pytest.mark.parametrize(
        "ch",
        [
            sy.Channel(
                name="test",
                data_type=sy.DataType.FLOAT64,
            ),
            sy.Channel(
                name="test",
                rate=1 * sy.Rate.HZ,
            ),
        ],
    )
    @pytest.mark.channel
    def test_create_no_rate_or_index(self, client: sy.Synnax, ch: sy.Channel):
        """Should create a single valid channel"""
        with pytest.raises(sy.ValidationError):
            client.channels.create(ch)

    @pytest.mark.channel
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

    @pytest.mark.channel
    def test_retrieve_by_key_not_found(self, client: sy.Synnax):
        with pytest.raises(sy.QueryError):
            client.channels.retrieve(key="1-100000")

    @pytest.mark.channel
    def test_retrieve_by_node_id(
        self, two_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        res_channels = client.channels.retrieve(node_id=1)
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.node_id == 1

    @pytest.mark.channel
    def test_retrieve_by_name(
        self, two_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        res_channels = client.channels.retrieve(names=["test", "test2"])
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.name in ["test", "test2"]
