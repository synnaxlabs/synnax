#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from types import MappingProxyType
from typing import cast

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from synnax.framer.adapter import ReadFrameAdapter, WriteFrameAdapter
from synnax.framer.frame import FramePayload
from synnax.telem import CrudeSeries
from x.strings import random_name


@pytest.mark.framer
@pytest.mark.telem
class TestFrame:
    def test_construction_from_cols_and_series(self) -> None:
        f = sy.Frame(["big", "red", "dog"], [np.array([1, 2, 3]), np.array([4, 5, 6])])
        assert f["big"][0] == 1

    def test_construction_from_dict(self) -> None:
        f = sy.Frame({1: sy.Series([1, 2, 3, 4]), 3: sy.Series([4, 5, 6, 7])})
        assert f[1][0] == 1

    def test_construction_from_mapping(self) -> None:
        """Should keep each channel's series when built from a non-dict mapping."""
        f = sy.Frame(
            MappingProxyType({1: sy.Series([1, 2, 3, 4]), 3: sy.Series([4, 5, 6, 7])})
        )
        assert f.channels == [1, 3]
        assert list(f[1]) == [1, 2, 3, 4]
        assert list(f[3]) == [4, 5, 6, 7]

    def test_construction_from_data_frame(self) -> None:
        f = sy.Frame(
            pd.DataFrame(
                {"big": sy.Series([1, 2, 3, 4]), "dog": sy.Series([4, 5, 6, 7])}
            )
        )
        assert f["dog"][0] == 4

    def test_construction_from_keys_and_series(self) -> None:
        f = sy.Frame({1: sy.Series([1, 2, 3, 4]), 2: sy.Series([4, 5, 6, 7])})
        assert f[1][0] == 1

    def test_construction_from_frame(self) -> None:
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        f2 = sy.Frame(f)
        assert f is not f2
        assert f.series[0] is f2.series[0]

    def test_construction_from_payload(self) -> None:
        pld = FramePayload([1, 2], [sy.Series([1, 2, 3]), sy.Series([4, 5, 6])])
        f = sy.Frame(pld)
        assert f[1][1] == 2

    def test_pandas_interop(self) -> None:
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        mask = f["big"] > 1
        assert len(f[mask]["big"]) == 3

    def test_contains_name(self) -> None:
        """Should return True if the name is in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        assert "big" in f

    def test_contains_key(self) -> None:
        """Should return True if the key is in the Frame"""
        f = sy.Frame({122: sy.Series([1, 2, 3, 4])})
        assert 122 in f

    def test_not_contains_name(self) -> None:
        """Should return False if the name is not in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        assert "small" not in f

    def test_get_by_known_name(self) -> None:
        """Should find a key column by the channel name the frame knows."""
        f = sy.Frame({1: sy.Series([1, 2]), 2: sy.Series([3, 4])})
        f.names = {1: "big", 2: "dog"}
        assert list(f["dog"]) == [3, 4]
        assert "dog" in f
        assert "cat" not in f

    def test_get_by_known_key(self) -> None:
        """Should find a name column by the channel key the frame knows."""
        f = sy.Frame({"big": sy.Series([1, 2])})
        f.names = {1: "big"}
        assert list(f[1]) == [1, 2]
        assert 1 in f

    def test_get_by_shared_name(self) -> None:
        """Should raise when two key columns share the name."""
        f = sy.Frame({1: sy.Series([1, 2]), 2: sy.Series([3, 4])})
        f.names = {1: "big", 2: "big"}
        with pytest.raises(sy.MultipleFoundError):
            f["big"]
        assert list(f[2]) == [3, 4]

    def test_get_by_key_with_shared_name(self) -> None:
        """Should not match a name column whose name two channels share."""
        f = sy.Frame([1, "big"], [sy.Series([1, 2]), sy.Series([3, 4])])
        f.names = {1: "big", 2: "big"}
        assert list(f[1]) == [1, 2]

    def test_getitem_missing(self) -> None:
        """Should return an empty MultiSeries for a channel not in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        assert len(f["small"]) == 0

    def test_get(self) -> None:
        """Should return the series of a channel in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        series = f.get("big")
        assert series is not None
        assert list(series) == [1, 2, 3, 4]

    def test_get_missing(self) -> None:
        """Should return None for a channel not in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        assert f.get("small") is None

    def test_get_missing_default(self) -> None:
        """Should return the default for a channel not in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        default = sy.MultiSeries([sy.Series([5])])
        assert f.get("small", default) is default


@pytest.mark.framer
@pytest.mark.telem
class TestFrameToDF:
    def test_name_columns(self) -> None:
        """Should use the channel names of a name-keyed frame as the columns."""
        f = sy.Frame({"big": sy.Series([1, 2]), "dog": sy.Series([3, 4])})
        df = f.to_df()
        assert list(df.columns) == ["big", "dog"]
        assert list(df["dog"]) == [3, 4]

    def test_known_names_of_keys(self) -> None:
        """Should use the channel names the frame knows for key columns."""
        f = sy.Frame({1: sy.Series([1, 2]), 2: sy.Series([3, 4])})
        f.names = {1: "big", 2: "dog"}
        df = f.to_df()
        assert list(df.columns) == ["big", "dog"]
        assert list(df["big"]) == [1, 2]

    def test_key_fallback(self) -> None:
        """Should use the channel key when the frame does not know the name."""
        f = sy.Frame({1: sy.Series([1, 2]), 2: sy.Series([3, 4])})
        f.names = {2: "dog"}
        assert list(f.to_df().columns) == [1, "dog"]

    def test_shared_name_uses_keys(self) -> None:
        """Should use the channel keys for channels that share a name."""
        f = sy.Frame(
            {1: sy.Series([1]), 2: sy.Series([2]), 3: sy.Series([3])},
        )
        f.names = {1: "big", 2: "big", 3: "dog"}
        df = f.to_df()
        assert list(df) == [1, 2, "dog"]
        assert list(df[2]) == [2]

    def test_channel_order(self) -> None:
        """Should order the columns by the first appearance of each channel."""
        f = sy.Frame(
            [30, 10, 20, 10],
            [
                sy.Series([1, 2]),
                sy.Series([3]),
                sy.Series([5, 6]),
                sy.Series([4]),
            ],
        )
        df = f.to_df()
        assert list(df) == [30, 10, 20]
        assert list(df[10]) == [3, 4]

    def test_append_merges_names(self) -> None:
        """Should keep the channel names of an appended frame."""
        f = sy.Frame()
        other = sy.Frame({1: sy.Series([1, 2])})
        other.names = {1: "big"}
        f.append(other)
        assert list(f.to_df().columns) == ["big"]


@pytest.mark.framer
class TestWriteFrameAdapter:
    @pytest.fixture(scope="class")
    @classmethod
    def adapter(cls, client: sy.Synnax) -> tuple[WriteFrameAdapter, sy.Channel]:
        ch = client.channels.create(
            name=f"test_{random_name()}",
            leaseholder=1,
            virtual=True,
            data_type=sy.DataType.FLOAT64,
        )
        adapter = WriteFrameAdapter(retriever=client.channels._retriever)
        adapter.update(ch.name)

        return adapter, ch

    def test_adaptation_of_keys_frame(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt of a Frame keyed by channel key."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(
            sy.Frame([ch.key], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_frame(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt of a Frame keyed by channel name."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(
            sy.Frame([ch.name], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_series(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a first argument of a channel name and a second
        argument of a series."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(
            ch.name, sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_float(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a first argument of a channel name and a second
        argument of a float."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(ch.name, 1.0)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_int(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a first argument of a channel name and a second
        argument of an int."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(ch.name, 1)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_series(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a first argument of a channel name and a second
        argument of a series."""
        write_adapter, ch = adapter
        series: list[CrudeSeries] = [
            sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)
        ]
        o = write_adapter.adapt([ch.name], series)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_dict_series(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a dict of channel names to series."""
        write_adapter, ch = adapter
        o = write_adapter.adapt({ch.name: sy.Series([1, 2, 3])})
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_non_dict_mapping(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should adapt a non-dict mapping of channel names to series."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(MappingProxyType({ch.name: [1.0, 2.0]}))
        assert o.channels == [ch.key]
        assert list(o.series[0]) == [1.0, 2.0]

    def test_adaptation_of_dict_float(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a dict of channel names to floats."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(
            {
                ch.name: 1.0,
            }
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_dict_timestamp(self, client: sy.Synnax) -> None:
        """It should correctly adapt a dict of channel names to timestamps."""
        ch = client.channels.create(
            sy.Channel(
                name=f"test_{random_name()}",
                virtual=True,
                data_type=sy.DataType.TIMESTAMP,
            )
        )
        adapter = WriteFrameAdapter(retriever=client.channels._retriever)
        adapter.update(ch.key)
        o = adapter.adapt({ch.name: sy.TimeStamp.now()})
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.TIMESTAMP

    def test_adaptation_of_channel_dict(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a dict of channels to series."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(
            {
                ch: 1.0,
            }
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_channel_payload(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """It should correctly adapt a FramePayload keyed by channel key."""
        write_adapter, ch = adapter
        o = write_adapter.adapt(ch, sy.TimeStamp.now())
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_multiple_payloads(
        self,
        adapter: tuple[WriteFrameAdapter, sy.Channel],
    ) -> None:
        """Should correctly adapt multiple channels and a single list of values"""
        write_adapter, ch = adapter
        o = write_adapter.adapt([ch], [1.0])
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64
        assert o.series[0][0] == 1.0

    def test_adaptation_of_list(
        self,
        adapter: tuple[WriteFrameAdapter, sy.Channel],
    ) -> None:
        """Should correctly adapt a channel and a list of values"""
        write_adapter, ch = adapter
        o = write_adapter.adapt(ch, [1.0, 2.0, 3.0])
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64
        assert len(o.series[0]) == 3

    def test_adaptation_of_multiple_series_and_single_payload(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """Should raise a validation error when there are more series than channels"""
        write_adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            series: list[CrudeSeries] = [[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]]
            write_adapter.adapt(ch, series)

    def test_adaptation_of_single_channel_and_no_data(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """Should raise a validation error when there are no series"""
        write_adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            write_adapter.adapt(ch)

    def test_adaptation_of_multiple_channels_and_no_data(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """Should raise a validation error when there are no series"""
        write_adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            write_adapter.adapt([ch, ch])

    def test_mismatch_of_channels_and_series_length(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """Should raise a validation error when there are more channels than series"""
        write_adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            write_adapter.adapt([ch, ch], [1.0])

    def test_validation_error_when_frame_with_nonexistent_channel_name_is_adapted(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """Should raise a validation error when a Frame with a nonexistent channel
        key is adapted
        """
        write_adapter, _ = adapter
        with pytest.raises(sy.ValidationError):
            write_adapter.adapt(
                sy.Frame(
                    ["caramela"], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)]
                )
            )

    def test_type_error_when_invalid_value_is_adapted(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """Should raise a type error when an invalid value is adapted"""
        write_adapter, _ = adapter
        with pytest.raises(TypeError):
            write_adapter.adapt(cast(sy.Frame, Exception("Invalid value")))

    def test_adapt_dict_keys(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ) -> None:
        """Should raise a type error when an invalid value is adapted"""
        write_adapter, ch = adapter
        adapted = write_adapter.adapt_dict_keys({ch.name: 123})
        assert adapted[ch.key] == 123

    def test_adapt_single_string(self, client: sy.Synnax) -> None:
        """Should correctly adapt a single string into a string based series"""
        ch = client.channels.create(
            name=f"test_{random_name()}",
            virtual=True,
            data_type=sy.DataType.STRING,
        )
        adapter = WriteFrameAdapter(retriever=client.channels._retriever)
        adapter.update(ch.name)
        adapted = adapter.adapt({ch.name: "hello"})
        assert adapted[ch.key][0] == "hello"

    def test_adapt_single_string_name_value_pair(self, client: sy.Synnax) -> None:
        ch = client.channels.create(
            name=f"test_{random_name()}",
            virtual=True,
            data_type=sy.DataType.STRING,
        )
        adapter = WriteFrameAdapter(retriever=client.channels._retriever)
        adapter.update(ch.name)
        adapted = adapter.adapt(ch.name, "hello")
        assert adapted[ch.key][0] == "hello"


@pytest.mark.framer
class TestReadFrameAdapter:
    """Comprehensive tests for ReadFrameAdapter functionality."""

    @pytest.fixture(scope="class")
    @classmethod
    def channels(cls, client: sy.Synnax) -> tuple[sy.Channel, sy.Channel, sy.Channel]:
        """Create three test channels for adapter testing."""
        ch1 = client.channels.create(
            name=f"test_read_adapter_ch1_{random_name()}",
            virtual=True,
            data_type=sy.DataType.FLOAT64,
        )
        ch2 = client.channels.create(
            name=f"test_read_adapter_ch2_{random_name()}",
            virtual=True,
            data_type=sy.DataType.INT64,
        )
        ch3 = client.channels.create(
            name=f"test_read_adapter_ch3_{random_name()}",
            virtual=True,
            data_type=sy.DataType.FLOAT32,
        )
        return ch1, ch2, ch3

    def test_initialization(self, client: sy.Synnax) -> None:
        """Should successfully initialize a ReadFrameAdapter."""
        adapter = ReadFrameAdapter(client.channels._retriever)
        assert adapter.retriever is not None
        assert adapter.keys == []
        assert adapter.codec is not None

    def test_update_with_keys(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should update adapter with channel keys without creating name mapping."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.key, ch2.key])

        # Should store keys directly
        assert adapter.keys == [ch1.key, ch2.key]

    def test_update_with_names(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should update adapter with channel names and create key->name mapping."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Should store keys (not names)
        assert adapter.keys == [ch1.key, ch2.key]

    def test_update_with_nonexistent_channel(self, client: sy.Synnax) -> None:
        """Should raise NotFoundError when updating with a nonexistent channel name."""
        adapter = ReadFrameAdapter(client.channels._retriever)
        with pytest.raises(
            sy.NotFoundError,
            match=r'channels not found matching \["nonexistent_channel_name_12345"\]',
        ):
            adapter.update(["nonexistent_channel_name_12345"])

    def test_adapt_with_keys_no_conversion(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should return frame unchanged when adapter was updated with keys."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.key, ch2.key])

        # Create a frame with keys
        original_frame = sy.Frame(
            [ch1.key, ch2.key],
            [
                sy.Series([1.0, 2.0, 3.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20, 30], data_type=sy.DataType.INT64),
            ],
        )

        adapted_frame = adapter.adapt(original_frame)

        # Frame should be returned unchanged
        assert adapted_frame.channels == [ch1.key, ch2.key]
        assert len(adapted_frame.series) == 2
        assert np.array_equal(adapted_frame[ch1.key], np.array([1.0, 2.0, 3.0]))
        assert np.array_equal(adapted_frame[ch2.key], np.array([10, 20, 30]))

    def test_adapt_with_names_conversion(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should convert channel keys to names when adapter was updated with names."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Create a frame with keys (as would come from server)
        frame_with_keys = sy.Frame(
            [ch1.key, ch2.key],
            [
                sy.Series([1.0, 2.0, 3.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20, 30], data_type=sy.DataType.INT64),
            ],
        )

        adapted_frame = adapter.adapt(frame_with_keys)

        # Keys should be converted to names
        assert adapted_frame.channels == [ch1.name, ch2.name]
        assert len(adapted_frame.series) == 2
        assert np.array_equal(adapted_frame[ch1.name], np.array([1.0, 2.0, 3.0]))
        assert np.array_equal(adapted_frame[ch2.name], np.array([10, 20, 30]))

    def test_adapt_filters_extra_channels(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should filter out channels not in the adapter mapping."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        # Only register ch1 and ch2
        adapter.update([ch1.name, ch2.name])

        # Create frame with all three channels
        frame_with_extra = sy.Frame(
            [ch1.key, ch2.key, ch3.key],
            [
                sy.Series([1.0, 2.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20], data_type=sy.DataType.INT64),
                sy.Series([100.0, 200.0], data_type=sy.DataType.FLOAT32),
            ],
        )

        adapted_frame = adapter.adapt(frame_with_extra)

        # Only ch1 and ch2 should remain
        assert len(adapted_frame.channels) == 2
        assert len(adapted_frame.series) == 2
        assert ch1.name in adapted_frame.channels
        assert ch2.name in adapted_frame.channels
        assert ch3.name not in adapted_frame.channels
        # Data should be preserved for kept channels
        assert np.array_equal(adapted_frame[ch1.name], np.array([1.0, 2.0]))
        assert np.array_equal(adapted_frame[ch2.name], np.array([10, 20]))

    def test_adapt_with_empty_frame(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should handle empty frames gracefully."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Create empty frame
        empty_frame = sy.Frame([], [])

        adapted_frame = adapter.adapt(empty_frame)

        assert len(adapted_frame.channels) == 0
        assert len(adapted_frame.series) == 0

    def test_adapt_with_no_matching_channels(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should return empty frame when no channels match the adapter mapping."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        # Register ch1 and ch2
        adapter.update([ch1.name, ch2.name])

        # Create frame with only ch3 (not in adapter)
        frame_no_match = sy.Frame(
            [ch3.key],
            [sy.Series([1.0, 2.0, 3.0], data_type=sy.DataType.FLOAT32)],
        )

        adapted_frame = adapter.adapt(frame_no_match)

        # All channels should be filtered out
        assert len(adapted_frame.channels) == 0
        assert len(adapted_frame.series) == 0

    def test_adapt_with_partial_matches(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should keep only matching channels and filter non-matching ones."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch3.name])  # Register ch1 and ch3

        # Frame contains ch1, ch2, ch3 but adapter only has ch1 and ch3
        frame_partial = sy.Frame(
            [ch1.key, ch2.key, ch3.key],
            [
                sy.Series([1.0, 2.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20], data_type=sy.DataType.INT64),
                sy.Series([100.0, 200.0], data_type=sy.DataType.FLOAT32),
            ],
        )

        adapted_frame = adapter.adapt(frame_partial)

        # Should only have ch1 and ch3
        assert len(adapted_frame.channels) == 2
        assert ch1.name in adapted_frame.channels
        assert ch3.name in adapted_frame.channels
        assert ch2.name not in adapted_frame.channels
        # Verify data integrity
        assert np.array_equal(adapted_frame[ch1.name], np.array([1.0, 2.0]))
        assert np.array_equal(adapted_frame[ch3.name], np.array([100.0, 200.0]))

    def test_adapt_preserves_series_data_types(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should preserve series data types during adaptation."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        frame = sy.Frame(
            [ch1.key, ch2.key],
            [
                sy.Series([1.0, 2.0, 3.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20, 30], data_type=sy.DataType.INT64),
            ],
        )

        adapted_frame = adapter.adapt(frame)

        # Data types should be preserved
        assert adapted_frame[ch1.name].data_type == sy.DataType.FLOAT64
        assert adapted_frame[ch2.name].data_type == sy.DataType.INT64

    def test_adapt_with_mixed_keys_and_names(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should handle frames with mixed keys and names (though unusual)."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Frame with mix of key and name (ch1 as key, ch2 as name)
        # This tests the isinstance(k, ChannelKey) check in adapt method
        frame_mixed = sy.Frame(
            [ch1.key, ch2.name],
            [
                sy.Series([1.0, 2.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20], data_type=sy.DataType.INT64),
            ],
        )

        adapted_frame = adapter.adapt(frame_mixed)

        # Both should be converted to names
        assert ch1.name in adapted_frame.channels
        # ch2.name should remain unchanged since it's already a name
        assert ch2.name in adapted_frame.channels

    def test_adapt_preserves_series_order(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should preserve the order of series when adapting."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name, ch3.name])

        frame = sy.Frame(
            [ch1.key, ch2.key, ch3.key],
            [
                sy.Series([1.0, 2.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20], data_type=sy.DataType.INT64),
                sy.Series([100.0, 200.0], data_type=sy.DataType.FLOAT32),
            ],
        )

        adapted_frame = adapter.adapt(frame)

        # Order should be preserved
        assert adapted_frame.channels[0] == ch1.name
        assert adapted_frame.channels[1] == ch2.name
        assert adapted_frame.channels[2] == ch3.name

    def test_codec_update(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should update codec with channel keys and data types."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)

        # Codec should be empty initially
        initial_codec = adapter.codec

        # Update with channels
        adapter.update([ch1.name, ch2.name])

        # Codec should have been updated
        assert adapter.codec is initial_codec  # Same instance
        # Keys should be registered in adapter
        assert ch1.key in adapter.keys
        assert ch2.key in adapter.keys

    def test_multiple_updates(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should handle multiple update calls correctly."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)

        # First update with ch1 and ch2
        adapter.update([ch1.name, ch2.name])
        assert len(adapter.keys) == 2

        # Second update with ch3 only
        adapter.update([ch3.name])
        assert len(adapter.keys) == 1
        assert adapter.keys == [ch3.key]

        # Adapt should now only work for ch3
        frame = sy.Frame(
            [ch1.key, ch2.key, ch3.key],
            [
                sy.Series([1.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10], data_type=sy.DataType.INT64),
                sy.Series([100.0], data_type=sy.DataType.FLOAT32),
            ],
        )

        adapted_frame = adapter.adapt(frame)
        assert len(adapted_frame.channels) == 1
        assert adapted_frame.channels[0] == ch3.name

    def test_update_switches_between_keys_and_names(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should correctly switch between key mode and name mode."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)

        # Start with names
        adapter.update([ch1.name, ch2.name])
        frame_names = sy.Frame(
            [ch1.key, ch2.key],
            [
                sy.Series([1.0, 2.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10, 20], data_type=sy.DataType.INT64),
            ],
        )
        adapted_names = adapter.adapt(frame_names)
        assert adapted_names.channels == [ch1.name, ch2.name]

        # Switch to keys
        adapter.update([ch1.key, ch2.key])
        frame_keys = sy.Frame(
            [ch1.key, ch2.key],
            [
                sy.Series([3.0, 4.0], data_type=sy.DataType.FLOAT64),
                sy.Series([30, 40], data_type=sy.DataType.INT64),
            ],
        )
        adapted_keys = adapter.adapt(frame_keys)
        # Should return unchanged (keys)
        assert adapted_keys.channels == [ch1.key, ch2.key]

    def test_adapt_records_names_with_keys(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ) -> None:
        """Should record the channel names in a key-keyed frame."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.key, ch2.key])
        frame = sy.Frame(
            [ch1.key, ch2.key],
            [
                sy.Series([1.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10], data_type=sy.DataType.INT64),
            ],
        )
        adapted = adapter.adapt(frame)
        assert adapted.channels == [ch1.key, ch2.key]
        assert list(adapted.to_df().columns) == [ch1.name, ch2.name]

    @pytest.mark.parametrize("by_name", [True, False])
    def test_adapt_sorts_into_requested_order(
        self,
        client: sy.Synnax,
        channels: tuple[sy.Channel, sy.Channel, sy.Channel],
        by_name: bool,
    ) -> None:
        """Should sort the series into the order of the requested channels."""
        ch1, ch2, ch3 = channels
        requested = [ch3, ch1, ch2]
        adapter = ReadFrameAdapter(client.channels._retriever)
        if by_name:
            adapter.update([ch.name for ch in requested])
        else:
            adapter.update([ch.key for ch in requested])
        frame = sy.Frame(
            [ch1.key, ch2.key, ch3.key],
            [
                sy.Series([1.0], data_type=sy.DataType.FLOAT64),
                sy.Series([10], data_type=sy.DataType.INT64),
                sy.Series([100.0], data_type=sy.DataType.FLOAT32),
            ],
        )
        adapted = adapter.adapt(frame)
        assert list(adapted.to_df().columns) == [ch.name for ch in requested]
        assert list(adapted.to_df()[ch3.name]) == [100.0]

    @pytest.mark.parametrize("by_name", [True, False])
    def test_adapt_keeps_first_position_of_repeated_channel(
        self,
        client: sy.Synnax,
        channels: tuple[sy.Channel, sy.Channel, sy.Channel],
        by_name: bool,
    ) -> None:
        """Should order a repeated channel by its first requested position."""
        ch1, ch2, _ = channels
        requested = [ch1, ch2, ch1]
        adapter = ReadFrameAdapter(client.channels._retriever)
        if by_name:
            adapter.update([ch.name for ch in requested])
        else:
            adapter.update([ch.key for ch in requested])
        frame = sy.Frame(
            [ch2.key, ch1.key],
            [
                sy.Series([10], data_type=sy.DataType.INT64),
                sy.Series([1.0], data_type=sy.DataType.FLOAT64),
            ],
        )
        adapted = adapter.adapt(frame)
        assert list(adapted.to_df().columns) == [ch1.name, ch2.name]
