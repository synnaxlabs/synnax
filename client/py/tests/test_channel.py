#  Copyright 2023 Synnax Labs, Inc.
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


@pytest.mark.channel
class TestChannel:
    """Tests all things related to channel operations. Create, delete, retrieve, etc."""

    @pytest.fixture(scope="class")
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

    def test_create_list(self, two_channels: list[sy.Channel]):
        """Should create a list of valid channels"""
        assert len(two_channels) == 2
        for channel in two_channels:
            assert channel.name.startswith("test")
            assert channel.key != ""

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

    def test_create_virtual(self, client: sy.Synnax):
        """Should create a virtual channel"""
        channel = client.channels.create(
            name="test", data_type=sy.DataType.JSON, virtual=True
        )
        res = client.channels.retrieve(channel.key)
        assert res.virtual is True

    def test_create_virtual_from_class(self, client: sy.Synnax):
        """Should create a virtual channel from the class"""
        channel = sy.Channel(
            name="test", 
            data_type=sy.DataType.JSON, 
            virtual=True
        )
        channel = client.channels.create(channel)
        res = client.channels.retrieve(channel.key)
        assert res.virtual is True

    def test_create_calculation_from_class(self, client: sy.Synnax):
        """Should create a calculation channel from the class"""
        base_v_channel = client.channels.create(
            name="test",
            data_type=sy.DataType.FLOAT32,
            virtual=True
        )
        channel = sy.Channel(
            name="test",
            data_type=sy.DataType.FLOAT32,
            expression="return 1 + 1",
            requires=[base_v_channel.key],
            virtual=True
        )
        channel = client.channels.create(channel)
        res = client.channels.retrieve(channel.key)
        assert res.expression == "return 1 + 1"

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
        self, two_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        """Should retrieve channels using a list of keys"""
        res_channels = client.channels.retrieve(
            [channel.key for channel in two_channels]
        )
        assert len(res_channels) == 2
        for i, channel in enumerate(res_channels):
            assert two_channels[i].key == channel.key
            assert isinstance(two_channels[i].data_type.density, sy.Density)

    def test_retrieve_by_key_not_found(self, client: sy.Synnax):
        """Should raise QueryError when key not found"""
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(1234)

    def test_retrieve_by_list_of_names(
        self, two_channels: list[sy.Channel], client: sy.Synnax
    ) -> None:
        """Should retrieve channels using list of names"""
        res_channels = client.channels.retrieve(["test", "test2"])
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.name in ["test", "test2"]

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
        self, client: sy.Synnax, two_channels: list[sy.channel]
    ):
        channels = client.channels.retrieve(
            [str(two_channels[0].key), str(two_channels[1].key)]
        )
        for channel in channels:
            assert channel.name in ["test", "test2"]

    def test_retrieve_bad_numeric_string(self, client: sy.Synnax):
        ch1 = client.channels.create(
            data_type=sy.DataType.FLOAT32, name="test1", rate=1 * sy.Rate.HZ
        )
        ch2 = client.channels.create(
            data_type=sy.DataType.FLOAT32, name=str(ch1.key), rate=1 * sy.Rate.HZ
        )

        # Should get first channel since the numeric string gets converted to a key
        result_channel = client.channels.retrieve(ch2.name)
        assert result_channel.name == "test1"

    def test_retrieve_single_multiple_found(
        self,
        client: sy.Synnax,
        two_channels: list[sy.Channel],
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
                    rate=1 * sy.Rate.HZ,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name="strange_channel_regex_2",
                    rate=1 * sy.Rate.HZ,
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
                    rate=1 * sy.Rate.HZ,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name=str(uuid.uuid4()),
                    rate=1 * sy.Rate.HZ,
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
                rate=1 * sy.Rate.HZ,
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
                rate=1 * sy.Rate.HZ,
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
                rate=1 * sy.Rate.HZ,
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
        new_names = [str(uuid.uuid4()), str(uuid.uuid4())]
        client.channels.rename([channel.key for channel in channels], new_names)
        for i, name in enumerate(new_names):
            retrieved = client.channels.retrieve(name)
            assert retrieved.name == name


class TestChannelRetriever:
    """Tests methods internal to the channel retriever that are not publicly availble
    through the ChannelClient.
    """

    def test_retrieve_one(self, client: sy.Synnax):
        ch = client.channels.create(
            data_type=sy.DataType.FLOAT32, name="test", rate=1 * sy.Rate.HZ
        )
        retrieved = client.channels._retriever.retrieve_one(ch.key)
        assert retrieved.key == ch.key

    def test_retrieve_one_not_found(self, client: sy.Synnax):
        with pytest.raises(sy.NotFoundError):
            client.channels._retriever.retrieve_one(1234)
