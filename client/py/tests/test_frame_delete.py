#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import pandas as pd
import pytest

import synnax as sy
from synnax.util.random import random_name
from tests.telem import seconds_linspace


@pytest.mark.framer
@pytest.mark.delete
class TestDeleter:
    def test_basic_delete(self, indexed_pair: sy.Channel, client: sy.Synnax):
        idx_ch, _ = indexed_pair
        with client.open_writer(sy.TimeSpan.SECOND * 1, idx_ch) as w:
            data = seconds_linspace(1, 50)
            w.write(pd.DataFrame({idx_ch.key: data}))
            w.commit()

        client.delete(
            [idx_ch.key],
            sy.TimeRange(0, sy.TimeStamp(25 * sy.TimeSpan.SECOND)),
        )

        data = idx_ch.read(sy.TimeRange.MAX)
        assert data.to_numpy().size == 26
        assert data.time_range == sy.TimeRange(
            sy.TimeStamp(25 * sy.TimeSpan.SECOND),
            sy.TimeStamp(50 * sy.TimeSpan.SECOND) + 1,
        )

    def test_delete_by_name(self, indexed_pair: list[sy.Channel], client: sy.Synnax):
        idx_ch, _ = indexed_pair
        with client.open_writer(sy.TimeSpan.SECOND * 1, idx_ch) as w:
            data = seconds_linspace(1, 50)
            w.write(pd.DataFrame({idx_ch.key: data}))
            w.commit()

        client.delete(
            [idx_ch.name],
            sy.TimeRange(0, sy.TimeStamp(25 * sy.TimeSpan.SECOND)),
        )

        data = idx_ch.read(sy.TimeRange.MAX)
        assert data.to_numpy().size == 26
        assert data.time_range == sy.TimeRange(
            sy.TimeStamp(25 * sy.TimeSpan.SECOND),
            sy.TimeStamp(50 * sy.TimeSpan.SECOND) + 1,
        )

    def test_delete_channel_not_found_name(
        self, indexed_pair: sy.Channel, client: sy.Synnax
    ):
        idx_ch, _ = indexed_pair
        client.write(0, idx_ch.key, seconds_linspace(1, 50))
        with pytest.raises(sy.NotFoundError):
            client.delete([idx_ch.name, "kaka"], sy.TimeRange.MAX)

        data = idx_ch.read(sy.TimeRange.MAX)
        assert data.size == 50 * 8

    def test_delete_channel_not_found_key(
        self, indexed_pair: sy.Channel, client: sy.Synnax
    ):
        idx_ch, _ = indexed_pair
        client.write(0, idx_ch.key, seconds_linspace(1, 50))
        with pytest.raises(sy.NotFoundError):
            client.delete([idx_ch.key, 1234], sy.TimeRange.MAX)

        data = idx_ch.read(sy.TimeRange.MAX)
        assert data.size == 50 * 8

    def test_delete_with_writer(
        self, indexed_pair: list[sy.Channel], client: sy.Synnax
    ):
        idx_ch, _ = indexed_pair
        with client.open_writer(0, idx_ch.key):
            with pytest.raises(sy.UnauthorizedError):
                client.delete(
                    [idx_ch.key],
                    sy.TimeRange(
                        sy.TimeStamp(1 * sy.TimeSpan.SECOND).range(
                            sy.TimeStamp(2 * sy.TimeSpan.SECOND)
                        )
                    ),
                )

    def test_delete_index_alone(self, client: sy.Synnax):
        ch1 = client.channels.create(
            sy.Channel(
                name=random_name(), data_type=sy.DataType.TIMESTAMP, is_index=True
            )
        )

        ch2 = sy.Channel(
            name=random_name(),
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
        ch2.write(0, np.array([0, 1, 2, 3, 4], dtype=np.float32))

        with pytest.raises(Exception):
            client.delete(
                [ch1.key],
                sy.TimeRange(
                    sy.TimeStamp(1 * sy.TimeSpan.SECOND).range(
                        sy.TimeStamp(2 * sy.TimeSpan.SECOND)
                    )
                ),
            )
