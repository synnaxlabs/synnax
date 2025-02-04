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
from synnax.framer.adapter import WriteFrameAdapter
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
    def adapter(self, client: sy.Synnax) -> [WriteFrameAdapter, sy.Channel]:
        ch = client.channels.create(
            name=f"test-{random.randint(0, 100000)}",
            leaseholder=1,
            rate=25 * sy.Rate.HZ,
            data_type=sy.DataType.FLOAT64,
        )
        adapter = WriteFrameAdapter(client.channels._retriever)
        adapter.update(ch.name)

        return adapter, ch

    def test_adaptation_of_keys_frame(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt of a Frame keyed by channel key."""
        adapter, ch = adapter
        o = adapter.adapt(
            Frame([ch.key], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_frame(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt of a Frame keyed by channel name."""
        adapter, ch = adapter
        o = adapter.adapt(
            Frame([ch.name], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_series(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a series."""
        adapter, ch = adapter
        o = adapter.adapt(ch.name, sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64))
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_float(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a float."""
        adapter, ch = adapter
        o = adapter.adapt(ch.name, 1.0)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_int(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a first argument of a channel name and a second
        argument of an int."""
        adapter, ch = adapter
        o = adapter.adapt(ch.name, 1)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_series(self, adapter: [WriteFrameAdapter, sy.Channel]):
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

    def test_adaptation_of_dict_series(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a dict of channel names to series."""
        adapter, ch = adapter
        o = adapter.adapt({ch.name: sy.Series([1, 2, 3])})
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_dict_float(self, adapter: [WriteFrameAdapter, sy.Channel]):
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
                name=f"test-{random.randint(0, 100000)}",
                rate=25 * sy.Rate.HZ,
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

    def test_adaptation_of_channel_dict(self, adapter: [WriteFrameAdapter, sy.Channel]):
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
        self, adapter: [WriteFrameAdapter, sy.Channel]
    ):
        """It should correctly adapt a FramePayload keyed by channel key."""
        adapter, ch = adapter
        o = adapter.adapt(ch, sy.TimeStamp.now())
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_multiple_payloads(
        self,
        adapter: [WriteFrameAdapter, sy.Channel],
    ):
        """Should correctly adapt multiple channels and a single list of values"""
        adapter, ch = adapter
        o = adapter.adapt([ch], [1.0])
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.FLOAT64
        assert o.series[0][0] == 1.0

    def test_adaptation_of_list(
        self,
        adapter: [WriteFrameAdapter, sy.Channel],
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
        adapter, ch = adapter
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
            name=f"test-{random.randint(0, 100000)}",
            virtual=True,
            data_type=sy.DataType.STRING,
        )
        adapter = WriteFrameAdapter(client.channels._retriever)
        adapter.update(ch.name)
        adapted = adapter.adapt({ch.name: "hello"})
        assert adapted[ch.key][0] == "hello"

    def test_adapt_single_string_name_value_pair(self, client):
        ch = client.channels.create(
            name=f"test-{random.randint(0, 100000)}",
            virtual=True,
            data_type=sy.DataType.STRING,
        )
        adapter = WriteFrameAdapter(client.channels._retriever)
        adapter.update(ch.name)
        adapted = adapter.adapt(ch.name, "hello")
        assert adapted[ch.key][0] == "hello"
