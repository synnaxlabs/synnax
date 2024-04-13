#  Copyright 2023 Synnax Labs, Inc.
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
        adapter, channel = adapter
        o = adapter.adapt(
            Frame([channel.key], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_frame(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt of a Frame keyed by channel name."""
        adapter, channel = adapter
        o = adapter.adapt(
            Frame([channel.name], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)])
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_series(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a series."""
        adapter, channel = adapter
        o = adapter.adapt(
            channel.name, sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_float(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a float."""
        adapter, channel = adapter
        o = adapter.adapt(channel.name, 1.0)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_name_int(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a first argument of a channel name and a second
        argument of an int."""
        adapter, channel = adapter
        o = adapter.adapt(channel.name, 1)
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adaptation_of_names_series(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a first argument of a channel name and a second
        argument of a series."""
        adapter, channel = adapter
        o = adapter.adapt(
            [channel.name], [sy.Series([1, 2, 3], data_type=sy.DataType.FLOAT64)]
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adapataion_of_dict_series(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a dict of channel names to series."""
        adapter, channel = adapter
        o = adapter.adapt(
            {
                channel.name: sy.Series([1, 2, 3]),
            }
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adapation_of_dict_float(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a dict of channel names to floats."""
        adapter, channel = adapter
        o = adapter.adapt(
            {
                channel.name: 1.0,
            }
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64

    def test_adapation_of_dict_timestamp(self, client: sy.Synnax):
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
        o = adapter.adapt(
            {
                ch.name: sy.TimeStamp.now(),
            }
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == ch.key
        assert o.series[0].data_type == sy.DataType.TIMESTAMP

    def test_adapation_of_channel_dict(self, adapter: [WriteFrameAdapter, sy.Channel]):
        """It should correctly adapt a dict of channels to series."""
        adapter, channel = adapter
        o = adapter.adapt(
            {
                channel: 1.0,
            }
        )
        assert len(o.channels) == 1
        assert len(o.series) == 1
        assert o.channels[0] == channel.key
        assert o.series[0].data_type == sy.DataType.FLOAT64
