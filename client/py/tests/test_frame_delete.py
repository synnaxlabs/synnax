#  Copyright 2025 Synnax Labs, Inc.
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
import pandas as pd

@pytest.mark.framer
class TestDeleter:
    def test_basic_delete(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(0, channel.key) as w:
            data = np.random.rand(51).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.commit()

        client.delete(
            [channel.key],
            sy.TimeRange(0, sy.TimeStamp(1 * sy.TimeSpan.SECOND)),
        )

        data = channel.read(sy.TimeRange.MAX)
        assert data.to_numpy().size == 26
        assert data.time_range == sy.TimeRange(
            sy.TimeStamp(1 * sy.TimeSpan.SECOND),
            sy.TimeStamp(2 * sy.TimeSpan.SECOND) + 1
        )

    def test_delete_by_name(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(0, channel.key) as w:
            data = np.random.rand(51).astype(np.float64)
            w.write(pd.DataFrame({channel.key: data}))
            w.commit()

        client.delete(
            channels=[channel.name],
            tr=sy.TimeRange(0, sy.TimeStamp(1 * sy.TimeSpan.SECOND)),
        )

        data = channel.read(sy.TimeRange.MAX)
        assert data.size == 26 * 8
        assert data.time_range == sy.TimeRange(
            sy.TimeStamp(1 * sy.TimeSpan.SECOND),
            sy.TimeStamp(2 * sy.TimeSpan.SECOND) + 1
        )

    def test_delete_channel_not_found_name(
        self, channel: sy.Channel, client: sy.Synnax
    ):
        client.write(0, channel.key, np.random.rand(50).astype(np.float64))
        with pytest.raises(sy.NotFoundError):
            client.delete([channel.name, "kaka"], sy.TimeRange.MAX)

        data = channel.read(sy.TimeRange.MAX)
        assert data.size == 50 * 8

    def test_delete_channel_not_found_key(self, channel: sy.Channel, client: sy.Synnax):
        client.write(0, channel.key, np.random.rand(50).astype(np.float64))
        with pytest.raises(sy.NotFoundError):
            client.delete([channel.key, 23423], sy.TimeRange.MAX)

        data = channel.read(sy.TimeRange.MAX)
        assert data.size == 50 * 8

    def test_delete_with_writer(self, channel: sy.Channel, client: sy.Synnax):
        with client.open_writer(0, channel.key):
            with pytest.raises(sy.UnauthorizedError):
                client.delete(
                    [channel.key],
                    sy.TimeRange(
                        sy.TimeStamp(1 * sy.TimeSpan.SECOND).range(
                            sy.TimeStamp(2 * sy.TimeSpan.SECOND)
                        )
                    ),
                )

    def test_delete_index_alone(self, client: sy.Synnax):
        ch1 = client.channels.create(
            sy.Channel(name="index", data_type=sy.DataType.TIMESTAMP, is_index=True)
        )

        ch2 = sy.Channel(
            name="data",
            data_type=sy.DataType.FLOAT32,
            index=ch1.key,
        )

        ch2 = client.channels.create(ch2)
        timestamps = [
            sy.TimeStamp(0),
            sy.TimeStamp(1 * sy.TimeSpan.SECOND),
            sy.TimeStamp(2 * sy.TimeSpan.SECOND),
            sy.TimeStamp(3 * sy.TimeSpan.SECOND),
            sy.TimeStamp(4 * sy.TimeSpan.SECOND),
        ]
        ch1.write(0, np.array(timestamps))
        ch2.write(0, np.array([0, 1, 2, 3, 4]))

        with pytest.raises(Exception):
            client.delete(
                [ch1.key],
                sy.TimeRange(
                    sy.TimeStamp(1 * sy.TimeSpan.SECOND).range(
                        sy.TimeStamp(2 * sy.TimeSpan.SECOND))
                ),
            )

