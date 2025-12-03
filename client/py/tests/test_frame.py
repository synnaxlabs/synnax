#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from synnax.framer.adapter import ReadFrameAdapter, WriteFrameAdapter
from synnax.framer.frame import Frame, FramePayload


@pytest.mark.framer
@pytest.mark.telem
class TestFrame:
    def test_construction_from_cols_and_series(self):
        f = sy.Frame(["big", "red", "dog"], [np.array([1, 2, 3]), np.array([4, 5, 6])])
        assert f["big"][0] == 1

    def test_construction_from_dict(self):
        f = sy.Frame({1: sy.Series([1, 2, 3, 4]), 3: sy.Series([4, 5, 6, 7])})
        assert f[1][0] == 1

    def test_construction_from_data_frame(self):
        f = sy.Frame(
            pd.DataFrame(
                {"big": sy.Series([1, 2, 3, 4]), "dog": sy.Series([4, 5, 6, 7])}
            )
        )
        assert f["dog"][0] == 4

    def test_construction_from_keys_and_series(self):
        f = sy.Frame({1: sy.Series([1, 2, 3, 4]), 2: sy.Series([4, 5, 6, 7])})
        assert f[1][0] == 1

    def test_construction_from_frame(self):
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        f2 = sy.Frame(f)
        assert f is not f2
        assert f.series[0] is f2.series[0]

    def test_construction_from_payload(self):
        pld = FramePayload([1, 2], [sy.Series([1, 2, 3]), sy.Series([4, 5, 6])])
        f = sy.Frame(pld)
        assert f[1][1] == 2

    def test_pandas_interop(self):
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        assert len(f[f["big"] > 1]["big"]) == 3

    def test_contains_name(self):
        """Should return True if the name is in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        assert "big" in f

    def test_contains_key(self):
        """Should return True if the key is in the Frame"""
        f = sy.Frame({122: sy.Series([1, 2, 3, 4])})
        assert 122 in f

    def test_not_contains_name(self):
        """Should return False if the name is not in the Frame"""
        f = sy.Frame({"big": sy.Series([1, 2, 3, 4])})
        assert "small" not in f


@pytest.mark.framer
class TestWriteFrameAdapter:
    @pytest.fixture(scope="class")
    def adapter(self, client: sy.Synnax) -> tuple[WriteFrameAdapter, sy.Channel]:
        ch = client.channels.create(
            name=f"test_{random.randint(0, 100000)}",
            leaseholder=1,
            virtual=True,
            data_type=sy.DataType.FLOAT64,
        )
        adapter = WriteFrameAdapter(client.channels._retriever)
        adapter.update(ch.name)

        return adapter, ch

    def test_adaptation_of_keys_frame(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt of a Frame keyed by channel key."""
        adapter, ch = adapter
        o = adapter.adapt(
            Frame([ch.key], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_frame(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt of a Frame keyed by channel name."""
        adapter, ch = adapter
        o = adapter.adapt(
            Frame([ch.name], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_series(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a series."""
        adapter, ch = adapter
        o = adapter.adapt(ch.name, sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64))
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_float(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a float."""
        adapter, ch = adapter
        o = adapter.adapt(ch.name, 1.0)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_int(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a first argument of a channel name and a second
        argument of an int."""
        adapter, ch = adapter
        o = adapter.adapt(ch.name, 1)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_series(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a series."""
        adapter, ch = adapter
        o = adapter.adapt(
            [ch.name], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)]
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_dict_series(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a dict of channel names to series."""
        adapter, ch = adapter
        o = adapter.adapt({ch.name: sy.Series([1, 2, 3])})
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_dict_float(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a dict of channel names to floats."""
        adapter, ch = adapter
        o = adapter.adapt(
            {
                ch.name: 1.0,
            }
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_dict_timestamp(self, client: sy.Synnax):
        """It should correctly adapt a dict of channel names to timestamps."""
        ch = client.channels.create(
            sy.Channel(
                name=f"test_{random.randint(0, 100000)}",
                virtual=True,
                data_type=sy.DataType.TIMESTAMP,
            )
        )
        adapter = WriteFrameAdapter(client.channels._retriever)
        adapter.update(ch.key)
        o = adapter.adapt({ch.name: sy.TimeStamp.now()})
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.TIMESTAMP

    def test_adaptation_of_channel_dict(
        self, adapter: tuple[WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a dict of channels to series."""
        adapter, ch = adapter
        o = adapter.adapt(
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
    ):
        """It should correctly adapt a FramePayload keyed by channel key."""
        adapter, ch = adapter
        o = adapter.adapt(ch, sy.TimeStamp.now())
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_multiple_payloads(
        self,
        adapter: tuple[WriteFrameAdapter, sy.Channel],
    ):
        """Should correctly adapt multiple channels and a single list of values"""
        adapter, ch = adapter
        o = adapter.adapt([ch], [1.0])
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64
        assert o.series[0][0] == 1.0

    def test_adaptation_of_list(
        self,
        adapter: tuple[WriteFrameAdapter, sy.Channel],
    ):
        """Should correctly adapt a channel and a list of values"""
        adapter, ch = adapter
        o = adapter.adapt(ch, [1.0, 2.0, 3.0])
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64
        assert len(o.series[0]) == 3

    def test_adaptation_of_multiple_series_and_single_payload(self, adapter):
        """Should raise a validation error when there are more series than channels"""
        adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            adapter.adapt(ch, [[1.0, 2.0, 3.0], [4.0, 5.0, 6.0]])

    def test_adaptation_of_single_channel_and_no_data(self, adapter):
        """Should raise a validation error when there are no series"""
        adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            adapter.adapt(ch)

    def test_adaptation_of_multiple_channels_and_no_data(self, adapter):
        """Should raise a validation error when there are no series"""
        adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            adapter.adapt([ch, ch])

    def test_mismatch_of_channels_and_series_length(self, adapter):
        """Should raise a validation error when there are more channels than series"""
        adapter, ch = adapter
        with pytest.raises(sy.ValidationError):
            adapter.adapt([ch, ch], [1.0])

    def test_validation_error_when_frame_with_nonexistent_channel_name_is_adapted(
        self, adapter
    ):
        """Should raise a validation error when a Frame with a nonexistent channel
        key is adapted
        """
        adapter, _ = adapter
        with pytest.raises(sy.ValidationError):
            adapter.adapt(
                Frame(
                    ["caramela"], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)]
                )
            )

    def test_type_error_when_invalid_value_is_adapted(self, adapter):
        """Should raise a type error when an invalid value is adapted"""
        adapter, channel = adapter
        with pytest.raises(TypeError):
            adapter.adapt(Exception("Invalid value"))

    def test_adapt_dict_keys(self, adapter):
        """Should raise a type error when an invalid value is adapted"""
        adapter, ch = adapter
        adapted = adapter.adapt_dict_keys({ch.name: 123})
        assert adapted[ch.key] == 123

    def test_adapt_single_string(self, client):
        """Should correctly adapt a single string into a string based series"""
        ch = client.channels.create(
            name=f"test_{random.randint(0, 100000)}",
            virtual=True,
            data_type=sy.DataType.STRING,
        )
        adapter = WriteFrameAdapter(client.channels._retriever)
        adapter.update(ch.name)
        adapted = adapter.adapt({ch.name: "hello"})
        assert adapted[ch.key][0] == "hello"

    def test_adapt_single_string_name_value_pair(self, client):
        ch = client.channels.create(
            name=f"test_{random.randint(0, 100000)}",
            virtual=True,
            data_type=sy.DataType.STRING,
        )
        adapter = WriteFrameAdapter(client.channels._retriever)
        adapter.update(ch.name)
        adapted = adapter.adapt(ch.name, "hello")
        assert adapted[ch.key][0] == "hello"


@pytest.mark.framer
class TestReadFrameAdapter:
    """Comprehensive tests for ReadFrameAdapter functionality."""

    @pytest.fixture(scope="class")
    def channels(self, client: sy.Synnax) -> tuple[sy.Channel, sy.Channel, sy.Channel]:
        """Create three test channels for adapter testing."""
        ch1 = client.channels.create(
            name=f"test_read_adapter_ch1_{random.randint(0, 100000)}",
            virtual=True,
            data_type=sy.DataType.FLOAT64,
        )
        ch2 = client.channels.create(
            name=f"test_read_adapter_ch2_{random.randint(0, 100000)}",
            virtual=True,
            data_type=sy.DataType.INT64,
        )
        ch3 = client.channels.create(
            name=f"test_read_adapter_ch3_{random.randint(0, 100000)}",
            virtual=True,
            data_type=sy.DataType.FLOAT32,
        )
        return ch1, ch2, ch3

    def test_initialization(self, client: sy.Synnax):
        """Should successfully initialize a ReadFrameAdapter."""
        adapter = ReadFrameAdapter(client.channels._retriever)
        assert adapter.retriever is not None
        assert adapter.keys == []
        assert adapter.codec is not None

    def test_update_with_keys(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ):
        """Should update adapter with channel keys without creating name mapping."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.key, ch2.key])

        # Should store keys directly
        assert adapter.keys == [ch1.key, ch2.key]

    def test_update_with_names(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ):
        """Should update adapter with channel names and create key->name mapping."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Should store keys (not names)
        assert adapter.keys == [ch1.key, ch2.key]

    def test_update_with_nonexistent_channel(self, client: sy.Synnax):
        """Should raise KeyError when updating with a nonexistent channel name."""
        adapter = ReadFrameAdapter(client.channels._retriever)
        with pytest.raises(KeyError):
            adapter.update(["nonexistent_channel_name_12345"])

    def test_adapt_with_keys_no_conversion(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ):
        """Should return frame unchanged when adapter was updated with keys."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.key, ch2.key])

        # Create a frame with keys
        original_frame = Frame(
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
    ):
        """Should convert channel keys to names when adapter was updated with names."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Create a frame with keys (as would come from server)
        frame_with_keys = Frame(
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
    ):
        """Should filter out channels not in the adapter mapping."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        # Only register ch1 and ch2
        adapter.update([ch1.name, ch2.name])

        # Create frame with all three channels
        frame_with_extra = Frame(
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
    ):
        """Should handle empty frames gracefully."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Create empty frame
        empty_frame = Frame([], [])

        adapted_frame = adapter.adapt(empty_frame)

        assert len(adapted_frame.channels) == 0
        assert len(adapted_frame.series) == 0

    def test_adapt_with_no_matching_channels(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ):
        """Should return empty frame when no channels match the adapter mapping."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        # Register ch1 and ch2
        adapter.update([ch1.name, ch2.name])

        # Create frame with only ch3 (not in adapter)
        frame_no_match = Frame(
            [ch3.key],
            [sy.Series([1.0, 2.0, 3.0], data_type=sy.DataType.FLOAT32)],
        )

        adapted_frame = adapter.adapt(frame_no_match)

        # All channels should be filtered out
        assert len(adapted_frame.channels) == 0
        assert len(adapted_frame.series) == 0

    def test_adapt_with_partial_matches(
        self, client: sy.Synnax, channels: tuple[sy.Channel, sy.Channel, sy.Channel]
    ):
        """Should keep only matching channels and filter non-matching ones."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch3.name])  # Register ch1 and ch3

        # Frame contains ch1, ch2, ch3 but adapter only has ch1 and ch3
        frame_partial = Frame(
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
    ):
        """Should preserve series data types during adaptation."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        frame = Frame(
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
    ):
        """Should handle frames with mixed keys and names (though unusual)."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name])

        # Frame with mix of key and name (ch1 as key, ch2 as name)
        # This tests the isinstance(k, ChannelKey) check in adapt method
        frame_mixed = Frame(
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
    ):
        """Should preserve the order of series when adapting."""
        ch1, ch2, ch3 = channels
        adapter = ReadFrameAdapter(client.channels._retriever)
        adapter.update([ch1.name, ch2.name, ch3.name])

        frame = Frame(
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
    ):
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
    ):
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
        frame = Frame(
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
    ):
        """Should correctly switch between key mode and name mode."""
        ch1, ch2, _ = channels
        adapter = ReadFrameAdapter(client.channels._retriever)

        # Start with names
        adapter.update([ch1.name, ch2.name])
        frame_names = Frame(
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
        frame_keys = Frame(
            [ch1.key, ch2.key],
            [
                sy.Series([3.0, 4.0], data_type=sy.DataType.FLOAT64),
                sy.Series([30, 40], data_type=sy.DataType.INT64),
            ],
        )
        adapted_keys = adapter.adapt(frame_keys)
        # Should return unchanged (keys)
        assert adapted_keys.channels == [ch1.key, ch2.key]
