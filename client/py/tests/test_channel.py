#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import pytest

import synnax as sy
from x import filter
from x.strings import random_name
from x.telem import seconds_linspace
from x.testutil import assert_eventually


def channel_name() -> str:
    return f"test_{random_name()}"


@pytest.mark.channel
class TestChannel:
    """Tests all things related to channel operations. Create, delete, retrieve, etc."""

    def test_create_index(self, client: sy.Synnax) -> None:
        """Should create an index channel."""
        name = channel_name()
        channel = client.channels.create(
            name=name, data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        assert channel.name == name
        assert channel.key != 0
        assert channel.data_type == sy.DataType.TIMESTAMP
        assert channel.is_index is True
        assert channel.index == channel.key

    def test_create_index_channel_bad_data_type(self, client: sy.Synnax) -> None:
        """Should raise a validation error when creating an index channel with a bad
        data type
        """
        with pytest.raises(sy.ValidationError):
            client.channels.create(
                name=channel_name(), data_type=sy.DataType.FLOAT64, is_index=True
            )

    def test_create_index_no_data_type_provided(self, client: sy.Synnax) -> None:
        """Should infer the data type as TimeStamp when creating an index channel
        without a data type
        """
        ch = client.channels.create(name=channel_name(), is_index=True)
        assert ch.data_type == sy.DataType.TIMESTAMP

    def test_create_index_no_name_provided(self, client: sy.Synnax) -> None:
        """Should raise a validation error when creating an index channel without a
        name
        """
        with pytest.raises(sy.ValidationError):
            client.channels.create(data_type=sy.DataType.TIMESTAMP, is_index=True)

    def test_create_indexed_pair(self, client: sy.Synnax) -> None:
        """Should create a channel with an index channel"""
        idx = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        data_name = channel_name()
        data = client.channels.create(
            name=data_name, data_type=sy.DataType.FLOAT64, index=idx.key
        )
        assert data.name == data_name
        assert data.key != 0
        assert data.data_type == sy.DataType.FLOAT64
        assert data.is_index is False
        assert data.index == idx.key

    def test_create_nonexistent_index(self, client: sy.Synnax) -> None:
        """Should raise a validation error when creating a channel with a non-existent
        index
        """
        with pytest.raises(sy.ValidationError):
            client.channels.create(
                name=channel_name(), data_type=sy.DataType.FLOAT64, index=12345678
            )

    def test_create_indexed_pair_no_name(self, client: sy.Synnax) -> None:
        """Should raise a validation error when creating a data channel with no name"""
        idx = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        with pytest.raises(sy.ValidationError):
            client.channels.create(data_type=sy.DataType.FLOAT64, index=idx.key)

    def test_create_indexed_pair_no_data_type(self, client: sy.Synnax) -> None:
        """Should raise a validation error when creating an index channel with no data
        type
        """
        idx = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        with pytest.raises(sy.ValidationError):
            client.channels.create(name=channel_name(), index=idx.key)

    def test_create_from_list(self, client: sy.Synnax) -> None:
        """Should create a list of valid channels"""
        ch_one = sy.Channel(
            name="test" + channel_name(),
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
        )
        ch_two = sy.Channel(
            name="test" + channel_name(),
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        channels = client.channels.create([ch_one, ch_two])
        assert len(channels) == 2
        for channel in channels:
            assert channel.name.startswith("test")
            assert channel.key != 0

    def test_create_from_single_instance(self, client: sy.Synnax) -> None:
        """Should create a single channel from a channel instance"""
        name = channel_name()
        channel = sy.Channel(
            name=name,
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
        )
        channel = client.channels.create(channel)
        assert channel.name == name
        assert channel.key != 0
        assert channel.data_type == sy.DataType.TIMESTAMP
        assert channel.is_index is True
        assert channel.index == channel.key

    def test_create_bool(self, client: sy.Synnax) -> None:
        """Should create a channel with a BOOLEAN data type"""
        idx = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        ch = client.channels.create(
            name=channel_name(), data_type=sy.DataType.BOOLEAN, index=idx.key
        )
        assert ch.data_type == sy.DataType.BOOLEAN
        res = client.channels.retrieve(ch.key)
        assert res.data_type == sy.DataType.BOOLEAN

    def test_create_virtual(self, client: sy.Synnax) -> None:
        """Should create a virtual channel"""
        channel = client.channels.create(
            name=channel_name(), data_type=sy.DataType.JSON, virtual=True
        )
        res = client.channels.retrieve(channel.key)
        assert res.virtual is True

    def test_create_virtual_from_class(self, client: sy.Synnax) -> None:
        """Should create a virtual channel from the class"""
        channel = sy.Channel(
            name=channel_name(), data_type=sy.DataType.JSON, virtual=True
        )
        channel = client.channels.create(channel)
        res = client.channels.retrieve(channel.key)
        assert res.virtual is True

    def test_create_calculation_from_class(self, client: sy.Synnax) -> None:
        """Should create a calculation channel from the class"""
        channel = sy.Channel(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            expression="return 1 + 1",
            virtual=True,
        )
        channel = client.channels.create(channel)
        res = client.channels.retrieve(channel.key)
        assert res.expression == "return 1 + 1"

    def test_create_calculation_from_kwargs(self, client: sy.Synnax) -> None:
        """Should create a calculated channel from kwargs and auto-set virtual to
        True
        """
        idx_ch = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )
        channel = client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            expression="return 1 + 1",
        )
        res = client.channels.retrieve(channel.key)
        assert res.expression == "return 1 + 1"

    @pytest.mark.multi_node
    def test_create_with_leaseholder(self, client: sy.Synnax) -> None:
        """Should create a channel with a leaseholder"""
        channel = client.channels.create(
            name=channel_name(), data_type=sy.DataType.JSON, leaseholder=2, virtual=True
        )
        res = client.channels.retrieve(channel.key)
        assert res.leaseholder == 2

    def test_create_with_leaseholder_not_found(self, client: sy.Synnax) -> None:
        """Should raise a QueryError when leaseholder not found"""
        with pytest.raises(sy.QueryError):
            client.channels.create(
                name=channel_name(),
                data_type=sy.DataType.JSON,
                leaseholder=1234,
                virtual=True,
            )

    def test_create_invalid_nptype(self, client: sy.Synnax) -> None:
        """Should throw a Validation Error when passing invalid numpy data type"""
        with pytest.raises(TypeError):
            client.channels.create(data_type=np.csingle)

    def test_retrieve_by_key(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should retrieve channels using a list of keys"""
        res_channels = client.channels.retrieve(
            [channel.key for channel in indexed_pair]
        )
        assert len(res_channels) == 2
        for i, channel in enumerate(res_channels):
            assert indexed_pair[i].key == channel.key
            assert isinstance(indexed_pair[i].data_type.density, sy.Density)

    def test_retrieve_by_key_not_found(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when key not found"""
        with pytest.raises(sy.NotFoundError, match="Channel matching 1234 not found"):
            client.channels.retrieve(1234)

    def test_retrieve_by_list_of_names(
        self, indexed_pair: tuple[sy.Channel, sy.Channel], client: sy.Synnax
    ) -> None:
        """Should retrieve channels using list of names"""
        names = [ch.name for ch in indexed_pair]
        res_channels = client.channels.retrieve(names)
        assert len(res_channels) >= 2
        for channel in res_channels:
            assert channel.name in names

    def test_retrieve_list_of_names_not_found(self, client: sy.Synnax) -> None:
        """Should retrieve an empty list when can't find channels"""
        fake_names = ["fake1", "fake2", "fake3"]
        results = client.channels.retrieve(fake_names)
        assert len(results) == 0

    def test_retrieve_list_of_keys_not_found(self, client: sy.Synnax) -> None:
        """Should retrieve an empty list when can't find channels"""
        fake_keys = [1234, 5781, 99123]
        assert client.channels.retrieve(fake_keys) == []

    def test_retrieve_numeric_string(
        self, client: sy.Synnax, indexed_pair: tuple[sy.Channel, sy.Channel]
    ) -> None:
        names = [ch.name for ch in indexed_pair]
        channels = client.channels.retrieve(
            [str(indexed_pair[0].key), str(indexed_pair[1].key)]
        )
        for channel in channels:
            assert channel.name in names

    def test_retrieve_bad_numeric_string(self, client: sy.Synnax) -> None:
        ch1 = client.channels.create(
            data_type=sy.DataType.FLOAT32, name=channel_name(), virtual=True
        )

        # Should get first channel since the numeric string gets converted to a key
        result_channel = client.channels.retrieve(str(ch1.key))
        assert result_channel.name == ch1.name

    def test_retrieve_single_multiple_found(
        self,
        client: sy.Synnax,
    ) -> None:
        """Should raise a MultipleFoundError when retrieving a single channel with
        multiple matches"""
        with pytest.raises(sy.MultipleFoundError):
            prefix = channel_name()
            client.channels.create(
                name=f"{prefix}1",
                virtual=True,
                data_type=sy.DataType.FLOAT64,
            )
            client.channels.create(
                name=f"{prefix}2",
                virtual=True,
                data_type=sy.DataType.FLOAT64,
            )
            client.channels.retrieve(f"{prefix}.*")

    def test_retrieve_by_regex(self, client: sy.Synnax) -> None:
        """Should retrieve channels test1 and test2 using a regex"""
        prefix = channel_name()
        client.channels.create(
            [
                sy.Channel(
                    name=f"{prefix}1",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name=f"{prefix}2",
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        res_channels = client.channels.retrieve([f"^{prefix}.*"])
        assert len(res_channels) >= 2

    def test_retrieve_by_name_and_regex(self, client: sy.Synnax) -> None:
        """Should retrieve the channels that a literal name or a regex selects"""
        prefix = channel_name()
        literal, a, b, _ = client.channels.create(
            [
                sy.Channel(name=f"{prefix}_literal", virtual=True, data_type="float32"),
                sy.Channel(name=f"{prefix}_p_a", virtual=True, data_type="float32"),
                sy.Channel(name=f"{prefix}_p_b", virtual=True, data_type="float32"),
                sy.Channel(name=f"{prefix}_other", virtual=True, data_type="float32"),
            ]
        )
        res = client.channels.retrieve([literal.name, f"{prefix}_p_.*"])
        assert sorted(ch.key for ch in res) == sorted([literal.key, a.key, b.key])

    def test_retrieve_where(self, client: sy.Synnax) -> None:
        """Should retrieve the channels that a where selects"""
        prefix = channel_name()
        a, _, c = client.channels.create(
            [
                sy.Channel(name=f"{prefix}_a", virtual=True, data_type="float32"),
                sy.Channel(name=f"{prefix}_b", virtual=True, data_type="float64"),
                sy.Channel(name=f"{prefix}_c", virtual=True, data_type="float32"),
            ]
        )
        res = client.channels.retrieve(
            where=sy.channel.Where(
                name=filter.String(regex=f"^{prefix}"),
                data_type=filter.Set[sy.DataType](eq=sy.DataType.FLOAT32),
            )
        )
        assert sorted(ch.key for ch in res) == sorted([a.key, c.key])

    def test_retrieve_where_and_filters(self, client: sy.Synnax) -> None:
        """Should keep the channels that the names, the where, and the filters all
        select"""
        prefix = channel_name()
        a, b, c = client.channels.create(
            [
                sy.Channel(name=f"{prefix}_a", virtual=True, data_type="float32"),
                sy.Channel(name=f"{prefix}_b", virtual=True, data_type="float64"),
                sy.Channel(name=f"{prefix}_c", virtual=True, data_type="float32"),
            ]
        )
        res = client.channels.retrieve(
            [a.name, b.name, c.name],
            where=sy.channel.Where(
                not_=sy.channel.Where(key=filter.Set[int](eq=c.key))
            ),
            data_types=["float32"],
        )
        assert [ch.key for ch in res] == [a.key]

    def test_delete_by_key(self, client: sy.Synnax) -> None:
        """Should delete channels using a list of keys"""
        channels = client.channels.create(
            [
                sy.Channel(
                    name=channel_name(),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name=channel_name(),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        keys = [channel.key for channel in channels]
        client.channels.delete(keys)
        assert client.channels.retrieve(keys) == []

    def test_delete_by_name(self, client: sy.Synnax) -> None:
        """Should delete channels using a list of names"""
        channels = client.channels.create(
            [
                sy.Channel(
                    name=channel_name(),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name=channel_name(),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        names = [channel.name for channel in channels]
        client.channels.delete(names)
        results = client.channels.retrieve(names)
        assert len(results) == 0

    def test_delete_and_recreate_with_same_name(self, client: sy.Synnax) -> None:
        """Should be able to delete, recreate, and then query a channel with the same
        name."""
        name = channel_name()
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

    def test_single_rename(self, client: sy.Synnax) -> None:
        """Should rename a single channel"""
        name = channel_name()
        channel = client.channels.create(
            sy.Channel(
                name=name,
                virtual=True,
                data_type=sy.DataType.FLOAT64,
            )
        )
        new_name = channel_name()
        client.channels.rename(channel.key, new_name)
        retrieved = client.channels.retrieve(new_name)
        assert retrieved.name == new_name
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(name)

    def test_multiple_rename(self, client: sy.Synnax) -> None:
        """Should rename multiple channels"""
        channels = client.channels.create(
            [
                sy.Channel(
                    name=channel_name(),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
                sy.Channel(
                    name=channel_name(),
                    virtual=True,
                    data_type=sy.DataType.FLOAT64,
                ),
            ]
        )
        new_names = [channel_name(), channel_name()]
        client.channels.rename([channel.key for channel in channels], new_names)
        for name in new_names:
            retrieved = client.channels.retrieve(name)
            assert retrieved.name == name

    @pytest.fixture(scope="class")
    @classmethod
    def hundred_channels(cls, client: sy.Synnax) -> list[sy.Channel]:
        data = [
            sy.Channel(
                name=f"sensor_{i + 1}_{channel_name()}",
                virtual=True,
                data_type=sy.DataType.FLOAT64,
                internal=True,
            )
            for i in range(100)
        ]
        return client.channels.create(data)

    def test_create_list(self, hundred_channels: list[sy.Channel]) -> None:
        """Should create a list of 100 valid channels"""
        assert len(hundred_channels) == 100
        for channel in hundred_channels:
            assert channel.name.startswith("sensor")
            assert channel.key != 0

    def test_retrieve_list(
        self, client: sy.Synnax, hundred_channels: list[sy.Channel]
    ) -> None:
        """Should retrieve a list of 100 valid channels"""
        names = [ch.name for ch in hundred_channels]
        res_channels = client.channels.retrieve(names)
        assert len(res_channels) == 100
        for channel in res_channels:
            assert channel.name.startswith("sensor")
            assert channel.key != 0
            assert isinstance(channel.data_type.density, sy.Density)

    def test_retrieve_zero_key_single(self, client: sy.Synnax) -> None:
        """Should retrieve a channel with a key of zero"""
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(0)

    def test_retrieve_zero_key_multiple(self, client: sy.Synnax) -> None:
        """Should retrieve no channels for keys of zero"""
        assert client.channels.retrieve([0, 0, 0]) == []

    def test_create_channel_with_avg_operation_duration(
        self, client: sy.Synnax
    ) -> None:
        """Should create a calculated channel with an avg operation over a duration"""

        idx_ch = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        base_channel = client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )

        operation = sy.channel.Operation(type="avg", duration=sy.TimeSpan.SECOND * 10)
        channel = sy.Channel(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {base_channel.name}",
            operations=[operation],
        )
        created = client.channels.create(channel)

        assert created.name == channel.name
        assert created.virtual is True
        assert created.operations is not None
        assert len(created.operations) == 1
        assert created.operations[0].type == "avg"
        assert created.operations[0].duration == sy.TimeSpan.SECOND * 10
        # reset_channel defaults to 0 when not set
        assert created.operations[0].reset_channel == 0

    def test_create_channel_with_min_operation_reset_channel(
        self, client: sy.Synnax
    ) -> None:
        """Should create a calculated channel with a min operation triggered by a reset
        channel
        """

        idx_ch = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        base_channel = client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )
        reset_channel = client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.UINT8,
            index=idx_ch.key,
        )

        operation = sy.channel.Operation(type="min", reset_channel=reset_channel.key)
        channel = sy.Channel(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {base_channel.name}",
            operations=[operation],
        )
        created = client.channels.create(channel)

        assert created.name == channel.name
        assert created.virtual is True
        assert created.operations is not None
        assert len(created.operations) == 1
        assert created.operations[0].type == "min"
        assert created.operations[0].reset_channel == reset_channel.key
        assert created.operations[0].duration == 0

    def test_create_channel_with_max_operation(self, client: sy.Synnax) -> None:
        """Should create a calculated channel with a max operation"""

        idx_ch = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        base_channel = client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )

        operation = sy.channel.Operation(
            type="max",
            duration=sy.TimeSpan.SECOND * 5,
        )
        channel = sy.Channel(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {base_channel.name}",
            operations=[operation],
        )
        created = client.channels.create(channel)

        assert created.name == channel.name
        assert created.operations is not None
        assert len(created.operations) == 1
        assert created.operations[0].type == "max"
        assert created.operations[0].duration == sy.TimeSpan.SECOND * 5

    def test_retrieve_channel_with_operations(self, client: sy.Synnax) -> None:
        """Should retrieve a channel and preserve its operations"""

        idx_ch = client.channels.create(
            name=channel_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
        )
        base_channel = client.channels.create(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            index=idx_ch.key,
        )

        operation = sy.channel.Operation(type="avg", duration=sy.TimeSpan.SECOND * 15)
        channel = sy.Channel(
            name=channel_name(),
            data_type=sy.DataType.FLOAT32,
            expression=f"return {base_channel.name}",
            operations=[operation],
        )
        created = client.channels.create(channel)

        # Retrieve and verify operations are preserved
        retrieved = client.channels.retrieve(created.key)
        assert retrieved.name == channel.name
        assert retrieved.operations is not None
        assert len(retrieved.operations) == 1
        assert retrieved.operations[0].type == "avg"
        assert retrieved.operations[0].duration == sy.TimeSpan.SECOND * 15


@pytest.mark.channel
class TestChannelRetrieveFilters:
    """Tests the filters accepted by the channel client's retrieve method."""

    @pytest.fixture(scope="class")
    @classmethod
    def prefix(cls) -> str:
        return channel_name()

    @pytest.fixture(scope="class")
    @classmethod
    def create_channels(
        cls, client: sy.Synnax, prefix: str
    ) -> tuple[sy.Channel, sy.Channel, sy.Channel]:
        idx = client.channels.create(name=f"{prefix}_idx", is_index=True)
        data = client.channels.create(
            name=f"{prefix}_data", index=idx.key, data_type=sy.DataType.FLOAT32
        )
        virt = client.channels.create(
            name=f"{prefix}_virt", virtual=True, data_type=sy.DataType.FLOAT64
        )
        return idx, data, virt

    @staticmethod
    def names(channels: tuple[sy.Channel, ...]) -> list[str]:
        return [ch.name for ch in channels]

    @staticmethod
    def keys(channels: list[sy.Channel]) -> set[int]:
        return {ch.key for ch in channels}

    def test_search_term(
        self,
        client: sy.Synnax,
        prefix: str,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should retrieve channels whose names match the search term"""

        def check() -> None:
            res = client.channels.retrieve(search_term=prefix)
            assert self.keys(list(create_channels)) <= self.keys(res)

        assert_eventually(check)

    def test_single_with_filter(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should apply filters to a single key or name"""
        virt = create_channels[2]
        assert client.channels.retrieve(virt.name, virtual=True).key == virt.key
        with pytest.raises(sy.NotFoundError):
            client.channels.retrieve(virt.key, virtual=False)

    def test_data_types(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should keep only channels with one of the data types"""
        res = client.channels.retrieve(
            self.names(create_channels), data_types=[sy.DataType.FLOAT32, "float64"]
        )
        assert self.keys(res) == {create_channels[1].key, create_channels[2].key}

    def test_not_data_types(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should drop channels with one of the data types"""
        res = client.channels.retrieve(
            self.names(create_channels), not_data_types=[sy.DataType.TIMESTAMP]
        )
        assert self.keys(res) == {create_channels[1].key, create_channels[2].key}

    def test_virtual(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should keep only virtual or only non-virtual channels"""
        names = self.names(create_channels)
        assert self.keys(client.channels.retrieve(names, virtual=True)) == {
            create_channels[2].key
        }
        assert self.keys(client.channels.retrieve(names, virtual=False)) == {
            create_channels[0].key,
            create_channels[1].key,
        }

    def test_is_index(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should keep only index or only non-index channels"""
        names = self.names(create_channels)
        assert self.keys(client.channels.retrieve(names, is_index=True)) == {
            create_channels[0].key
        }
        assert self.keys(client.channels.retrieve(names, is_index=False)) == {
            create_channels[1].key,
            create_channels[2].key,
        }

    def test_internal(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should keep only internal or only non-internal channels"""
        names = self.names(create_channels)
        assert client.channels.retrieve(names, internal=True) == []
        assert len(client.channels.retrieve(names, internal=False)) == 3

    def test_node_key(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should keep only channels leased by the node"""
        idx, data, _ = create_channels
        names = self.names((idx, data))
        res = client.channels.retrieve(names, node_key=idx.leaseholder)
        assert self.keys(res) == {idx.key, data.key}
        assert client.channels.retrieve(names, node_key=idx.leaseholder + 1) == []

    def test_limit_offset(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should page through the matching channels"""
        names = self.names(create_channels)
        first = client.channels.retrieve(names, limit=2)
        rest = client.channels.retrieve(names, limit=2, offset=2)
        assert len(first) == 2
        assert len(rest) == 1
        assert self.keys(first) | self.keys(rest) == self.keys(list(create_channels))

    def test_no_params(self, client: sy.Synnax) -> None:
        """Should retrieve a list of channels when no key or name is given"""
        assert len(client.channels.retrieve(limit=1)) == 1

    def test_range_key(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should set the alias of the channel in the range without caching it"""
        data = create_channels[1]
        rng = client.ranges.create(
            name=random_name(),
            time_range=sy.TimeStamp.now().span_range(sy.TimeSpan.SECOND),
        )
        rng.set_alias(data.key, "aliased")
        assert client.channels.retrieve(data.key, range_key=rng.key).alias == (
            "aliased"
        )
        assert client.channels.retrieve(data.name, range_key=rng.key).alias == (
            "aliased"
        )
        assert client.channels.retrieve(data.key).alias is None

    def test_range_key_search(self, client: sy.Synnax) -> None:
        """Should put the channels whose alias in the range matches the search term
        first, keeping only those that match the other filters
        """
        term = random_name()
        named = client.channels.create(
            name=f"{term}_named", data_type=sy.DataType.FLOAT32, virtual=True
        )
        aliased = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT32, virtual=True
        )
        excluded = client.channels.create(
            name=random_name(), data_type=sy.DataType.FLOAT64, virtual=True
        )
        rng = client.ranges.create(
            name=random_name(),
            time_range=sy.TimeStamp.now().span_range(sy.TimeSpan.SECOND),
        )
        rng.set_alias(aliased.key, f"{term}_alias")
        rng.set_alias(excluded.key, f"{term}_excluded")

        def check() -> None:
            res = client.channels.retrieve(
                search_term=term,
                range_key=rng.key,
                data_types=[sy.DataType.FLOAT32],
            )
            assert [(ch.key, ch.alias) for ch in res] == [
                (aliased.key, f"{term}_alias"),
                (named.key, None),
            ]

        assert_eventually(check)

    def test_include_status(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should attach the status of a channel that has one"""
        virt = create_channels[2]
        client.statuses.set(
            sy.Status(
                key=str(virt.ontology_id),
                name=virt.name,
                variant=sy.status.VARIANT_SUCCESS,
                message="ok",
            )
        )
        res = client.channels.retrieve(virt.key, include_status=True)
        assert res.status is not None
        assert res.status.message == "ok"
        assert res.status.details == {}
        assert client.channels.retrieve(virt.key).status is None

    def test_include_status_none(
        self,
        client: sy.Synnax,
        create_channels: tuple[sy.Channel, sy.Channel, sy.Channel],
    ) -> None:
        """Should leave the status empty for a channel that has none"""
        idx = create_channels[0]
        res = client.channels.retrieve([idx.key], include_status=True)
        assert len(res) == 1
        assert res[0].status is None


class TestChannelRetriever:
    """Tests methods internal to the channel retriever that are not publicly availble
    through the ChannelClient.
    """

    def test_retrieve_one(self, client: sy.Synnax) -> None:
        ch = client.channels.create(
            data_type=sy.DataType.FLOAT32,
            name=channel_name(),
            virtual=True,
        )
        retrieved = client.channels._retriever.retrieve_one(ch.key)
        assert retrieved.key == ch.key

    def test_retrieve_one_not_found(self, client: sy.Synnax) -> None:
        with pytest.raises(sy.NotFoundError):
            client.channels._retriever.retrieve_one(1234)

    def test_retrieve_one_multiple_found(self, client: sy.Synnax) -> None:
        """Should raise MultipleFoundError when a pattern matches two channels."""
        prefix = channel_name()
        for suffix in ("a", "b"):
            client.channels.create(
                data_type=sy.DataType.FLOAT32, name=f"{prefix}_{suffix}", virtual=True
            )
        with pytest.raises(
            sy.MultipleFoundError, match=f"Expected one Channel matching {prefix}_"
        ):
            client.channels._retriever.retrieve_one(f"{prefix}_.*")


@pytest.mark.framer
class TestChannelWriteRead:
    def test_write_read(self, client: sy.Synnax) -> None:
        """Should create a channel and write then read from it"""
        channel = client.channels.create(
            sy.Channel(
                name=channel_name(),
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
            )
        )
        d = seconds_linspace(1, count=10)
        start = 1 * sy.TimeSpan.SECOND
        channel.write(start, d)
        data = channel.read(start, start + len(d) * sy.TimeSpan.SECOND)
        assert data.time_range is not None
        assert data.time_range.start == start
        assert len(d) == len(data)
        assert data.time_range.end == (
            start + (len(d) - 1) * sy.TimeSpan.SECOND + sy.TimeSpan.NANOSECOND
        )
        assert np.array_equal(data, d)
