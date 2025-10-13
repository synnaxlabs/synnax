#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

import numpy as np
import pytest

import synnax as sy
from tests.telem import seconds_linspace


@pytest.mark.channel
class TestChannel:
    """Tests all things related to channel operations. Create, delete, retrieve, etc."""

    def test_create_index(self, client: sy.Synnax):
        """Should create an index channel."""
        channel = client.channels.create(
            name="Time", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        assert channel.name == "Time"
        assert channel.key != 0
        assert channel.data_type == sy.DataType.TIMESTAMP
        assert channel.is_index is True
        assert channel.index == channel.key

    def test_create_index_channel_bad_data_type(self, client: sy.Synnax):
        """Should raise a validation error when creating an index channel with a bad data type"""
        with pytest.raises(sy.ValidationError):
            client.channels.create(
                name="Time", data_type=sy.DataType.FLOAT64, is_index=True
            )

    def test_create_index_no_data_type_provided(self, client: sy.Synnax):
        """Should infer the data type as TimeStamp when creating an index channel without a data type"""
        ch = client.channels.create(name="Time", is_index=True)
        assert ch.data_type == sy.DataType.TIMESTAMP

    def test_create_index_no_name_provided(self, client: sy.Synnax):
        """Should raise a validation error when creating an index channel without a name"""
        with pytest.raises(sy.ValidationError):
            client.channels.create(data_type=sy.DataType.TIMESTAMP, is_index=True)

    def test_create_indexed_pair(self, client: sy.Synnax):
        """Should create a channel with an index channel"""
        idx = client.channels.create(
            name="Time", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data = client.channels.create(
            name="Data", data_type=sy.DataType.FLOAT64, index=idx.key
        )
        assert data.name == "Data"
        assert data.key != 0
        assert data.data_type == sy.DataType.FLOAT64
        assert data.is_index is False
        assert data.index == idx.key

    def test_create_nonexistent_index(self, client: sy.Synnax):
        """Should raise a validation error when creating a channel with a non-existent index"""
        with pytest.raises(sy.ValidationError):
            client.channels.create(
                name="Data", data_type=sy.DataType.FLOAT64, index=1234
            )

    def test_create_indexed_pair_no_name(self, client: sy.Synnax):
        """Should raise a validation error when creating a data channel with no name"""
        idx = client.channels.create(
            name="Time", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        with pytest.raises(sy.ValidationError):
            client.channels.create(data_type=sy.DataType.FLOAT64, index=idx.key)

    def test_create_indexed_pair_no_data_type(self, client: sy.Synnax):
        """Should raise a validation error when creating an index channel with no data type"""
        idx = client.channels.create(
            name="Time", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        with pytest.raises(sy.ValidationError):
            client.channels.create(name="Data", index=idx.key)

    def test_create_from_list(self, client: sy.Synnax):
        """Should create a list of valid channels"""
        ch_one = sy.Channel(
            name="test_osterone",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        ch_two = sy.Channel(
            name="test_ostertwo",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        channels = client.channels.create([ch_one, ch_two])
        assert len(channels) == 2
        for channel in channels:
            assert channel.name.startswith("test")
            assert channel.key != ""

    def test_create_from_single_instance(self, client: sy.Synnax):
        """Should create a single channel from a channel instance"""
        channel = sy.Channel(
            name="test",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        channel = client.channels.create(channel)
        assert channel.name == "test"
        assert channel.key != 0
        assert channel.data_type == sy.DataType.TIMESTAMP
        assert channel.is_index is True
        assert channel.index == channel.key

    def test_create_virtual(self, client: sy.Synnax):
        """Should create a virtual channel"""
        channel = client.channels.create(
            name="test", data_type=sy.DataType.JSON, virtual=True
        )
        res = client.channels.retrieve(channel.key)
        assert res.virtual is True

    def test_create_virtual_from_class(self, client: sy.Synnax):
        """Should create a virtual channel from the class"""
        channel = sy.Channel(name="test", data_type=sy.DataType.JSON, virtual=True)
        channel = client.channels.create(channel)
        res = client.channels.retrieve(channel.key)
        assert res.virtual is True

    def test_create_calculation_from_class(self, client: sy.Synnax):
        """Should create a calculation channel from the class"""
        idx_ch = client.channels.create(
            name="test", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        base_v_channel = client.channels.create(
            name="test",
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )
        channel = sy.Channel(
            name="test",
            data_type=sy.DataType.FLOAT32,
            expression="return 1 + 1",
            requires=[base_v_channel.key],
            virtual=True,
        )
        channel = client.channels.create(channel)
        res = client.channels.retrieve(channel.key)
        assert res.expression == "return 1 + 1"

    def test_create_calculation_from_kwargs(self, client: sy.Synnax):
        """Should create a calculated channel from kwargs and auto-set virtual to True"""
        idx_ch = client.channels.create(
            name="test", data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        base_v_channel = client.channels.create(
            name="test",
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )
        channel = client.channels.create(
            name="test",
            data_type=sy.DataType.FLOAT32,
            expression="return 1 + 1",
            requires=[base_v_channel.key],
        )
        res = client.channels.retrieve(channel.key)
        assert res.expression == "return 1 + 1"
        assert res.requires == [base_v_channel.key]

    @pytest.mark.multi_node
    def test_create_with_leaseholder(self, client: sy.Synnax):
        """Should create a channel with a leaseholder"""
        channel = client.channels.create(
            name="test", data_type=sy.DataType.JSON, leaseholder=2, virtual=True
        )
        res = client.channels.retrieve(channel.key)
        assert res.leaseholder == 2

    def test_create_with_leaseholder_not_found(self, client: sy.Synnax):
        """Should raise a QueryError when leaseholder not found"""
        with pytest.raises(sy.QueryError):
            client.channels.create(
                name="test", data_type=sy.DataType.JSON, leaseholder=1234, virtual=True
            )

    def test_create_invalid_nptype(self, client: sy.Synnax):
        """Should throw a Validation Error when passing invalid numpy data type"""
        with pytest.raises(TypeError):
            client.channels.create(data_type=np.csingle)

    def test_retrieve_by_key(
        self, indexed_pair: list[sy.Channel], client: sy.Synnax
    ) -> None:
        """Should retrieve channels using a list of keys"""
        res_channels = client.channels.retrieve(
            [channel.key for channel in indexed_pair]
        )
        assert len(res_channels) == 2
        for i, channel in enumerate(res_channels):
            assert indexed_pair[i].key == channel.key
            assert isinstance(indexed_pair[i].data_type.density, sy.Density)

    def test_retrieve_by_key_not_found(self, client: sy.Synnax):
        """Should raise QueryError when key not found"""
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(1234)

    def test_retrieve_by_list_of_names(
        self, indexed_pair: list[sy.Channel], client: sy.Synnax
    ) -> None:
        """Should retrieve channels using list of names"""
        names = [ch.name for ch in indexed_pair]
        res_channels = client.channels.retrieve(names)
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.name in names

    def test_retrieve_list_of_names_not_found(self, client: sy.Synnax):
        """Should retrieve an empty list when can't find channels"""
        fake_names = ["fake1", "fake2", "fake3"]
        results = client.channels.retrieve(fake_names)
        assert len(results) == 0

    def test_retrieve_list_of_keys_not_found(self, client: sy.Synnax):
        """Should retrieve an empty list when can't find channels"""
        fake_keys = [1234, 5781, 99123]
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(fake_keys)

    def test_retrieve_numeric_string(
        self, client: sy.Synnax, indexed_pair: list[sy.channel]
    ):
        names = [ch.name for ch in indexed_pair]
        channels = client.channels.retrieve(
            [str(indexed_pair[0].key), str(indexed_pair[1].key)]
        )
        for channel in channels:
            assert channel.name in names

    def test_retrieve_bad_numeric_string(self, client: sy.Synnax):
        ch1 = client.channels.create(
            data_type=sy.DataType.FLOAT32, name="test1", virtual=True
        )
        ch2 = client.channels.create(
            data_type=sy.DataType.FLOAT32, name=str(ch1.key), virtual=True
        )

        # Should get first channel since the numeric string gets converted to a key
        result_channel = client.channels.retrieve(ch2.name)
        assert result_channel.name == "test1"

    def test_retrieve_single_multiple_found(
        self,
        client: sy.Synnax,
        indexed_pair: list[sy.Channel],
    ):
        """Should raise QueryError when retrieving a single channel with
        multiple matches"""
        with pytest.raises(sy.MultipleFoundError):
            client.channels.retrieve("test.*")

    def test_retrieve_by_regex(self, client: sy.Synnax):
        """Should retrieve channels test1 and test2 using a regex"""
        ch1 = client.channels.create(
            [
                sy.Channel(
                    name="strange_channel_regex_1",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name="strange_channel_regex_2",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        res_channels = client.channels.retrieve(["^strange_channel_regex_"])
        assert len(res_channels) >= 2

    def test_delete_by_key(self, client: sy.Synnax):
        """Should delete channels using a list of keys"""
        channels = client.channels.create(
            [
                sy.Channel(
                    name="test",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name="test2",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        keys = [channel.key for channel in channels]
        client.channels.delete(keys)
        with pytest.raises(sy.QueryError):
            client.channels.retrieve(keys)

    def test_delete_by_name(self, client: sy.Synnax):
        """Should delete channels using a list of names"""
        channels = client.channels.create(
            [
                sy.Channel(
                    name=str(uuid.uuid4()),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name=str(uuid.uuid4()),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        names = [channel.name for channel in channels]
        client.channels.delete(names)
        results = client.channels.retrieve(names)
        assert len(results) == 0

    def test_delete_and_recreate_with_same_name(self, client: sy.Synnax):
        """Should be able to delete, recreate, and then query a channel with the same
        name."""
        name = str(uuid.uuid4())
        ch = client.channels.create(
            sy.Channel(
                name=name,
                virtual=True,
                data_type=sy.DataType.FLOAT64,
            ),
            retrieve_if_name_exists=True,
        )
        ch_retrieved = client.channels.retrieve(name)
        assert ch.key == ch_retrieved.key
        client.channels.delete(name)
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(ch.key)
        ch2 = client.channels.create(
            sy.Channel(
                name=name,
                virtual=True,
                data_type=sy.DataType.FLOAT64,
            ),
            retrieve_if_name_exists=True,
        )
        assert ch2.key != ch.key
        ch2_retrieved = client.channels.retrieve(name)
        assert ch2.key == ch2_retrieved.key
        all_channels = client.channels.retrieve([".*"])
        keys = [channel.key for channel in all_channels]
        assert ch2.key in keys

    def test_single_rename(self, client: sy.Synnax):
        """Should rename a single channel"""
        name = str(uuid.uuid4())
        channel = client.channels.create(
            sy.Channel(
                name=name,
                virtual=True,
                data_type=sy.DataType.FLOAT64,
            )
        )
        new_name = str(uuid.uuid4())
        client.channels.rename(channel.key, new_name)
        retrieved = client.channels.retrieve(new_name)
        assert retrieved.name == new_name
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(name)

    def test_multiple_rename(self, client: sy.Synnax):
        """Should rename multiple channels"""
        channels = client.channels.create(
            [
                sy.Channel(
                    name="test",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name="test2",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        new_names = [str(uuid.uuid4()), str(uuid.uuid4())]
        client.channels.rename([channel.key for channel in channels], new_names)
        for i, name in enumerate(new_names):
            retrieved = client.channels.retrieve(name)
            assert retrieved.name == name

    @pytest.fixture(scope="class")
    def hundred_channels(self, client: sy.Synnax) -> list[sy.Channel]:
        data = []
        for i in range(100):
            data.append(
                sy.Channel(
                    name=f"sensor_{i+1}_{str(uuid.uuid4())}",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                    internal=True,
                )
            )
        return client.channels.create(data)

    def test_create_list(self, hundred_channels: list[sy.Channel]):
        """Should create a list of 100 valid channels"""
        assert len(hundred_channels) == 100
        for channel in hundred_channels:
            assert channel.name.startswith("sensor")
            assert channel.key != ""

    def test_retrieve_list(self, client: sy.Synnax, hundred_channels: list[sy.Channel]):
        """Should retrieve a list of 100 valid channels"""
        names = [ch.name for ch in hundred_channels]
        res_channels = client.channels.retrieve(names)
        assert len(res_channels) == 100
        for channel in res_channels:
            assert channel.name.startswith("sensor")
            assert channel.key != ""
            assert isinstance(channel.data_type.density, sy.Density)

    def test_retrieve_zero_key_single(self, client: sy.Synnax):
        """Should retrieve a channel with a key of zero"""
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(0)

    def test_retrieve_zero_key_multiple(self, client: sy.Synnax):
        """Should retrieve a list of channels with a key of zero"""
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve([0, 0, 0])


class TestChannelRetriever:
    """Tests methods internal to the channel retriever that are not publicly availble
    through the ChannelClient.
    """

    def test_retrieve_one(self, client: sy.Synnax):
        ch = client.channels.create(
            data_type=sy.DataType.FLOAT32,
            name="test",
            virtual=True,
        )
        retrieved = client.channels._retriever.retrieve_one(ch.key)
        assert retrieved.key == ch.key

    def test_retrieve_one_not_found(self, client: sy.Synnax):
        with pytest.raises(sy.NotFoundError):
            client.channels._retriever.retrieve_one(1234)


@pytest.mark.framer
class TestChannelWriteRead:
    def test_write_read(self, client: sy.Synnax):
        """Should create a channel and write then read from it"""
        channel = client.channels.create(
            sy.Channel(
                name="test",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
            )
        )
        d = seconds_linspace(1, 10)
        start = 1 * sy.TimeSpan.SECOND
        channel.write(start, d)
        data = channel.read(start, (start + len(d)) * sy.TimeSpan.SECOND)
        assert data.time_range.start == start
        assert len(d) == len(data)
        assert data.time_range.end == start + (len(d) - 1) * sy.TimeSpan.SECOND + 1
        assert np.array_equal(data, d)
