#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import pandas as pd
from collections.abc import Iterator
import synnax as sy

SAMPLES_PER_CHANNEL = 1_000_000
SAMPLES_PER_CHANNEL_PER_ITERATION = 100_000
ITERATIONS = 10
CHANNEL_COUNT = 2
TOTAL_SAMPLES = SAMPLES_PER_CHANNEL * ITERATIONS * CHANNEL_COUNT

START_TIME = sy.TimeStamp.now()


class TestConfig:
    _channels: list[sy.Channel]
    _start_time = sy.TimeStamp.now()
    _cached_data: pd.DataFrame | None = None
    iterations = ITERATIONS
    channel_count = CHANNEL_COUNT
    samples_per_channel = SAMPLES_PER_CHANNEL

    def __init__(self):
        c = CHANNEL_COUNT
        c -= 1
        idx = sy.Channel(name="timestamps", data_type="timestamp", is_index=True)
        self._channels = [idx]
        for i in range(c):
            ch = sy.Channel(name=f"channel_{i}", data_type="float32", index=idx.key)
            self._channels.append(ch)
        self._start_time = sy.TimeStamp.now()

    def re_assign_indexes(self):
        for i in range(1, len(self._channels)):
            self._channels[i].index = self._channels[0].key

    @property
    def start(self) -> sy.TimeStamp:
        return self._start_time

    @property
    def channels(self) -> list[sy.Channel]:
        return self._channels

    def frames(self, index: bool = False) -> Iterator[pd.DataFrame]:
        start = self._start_time
        end = start + (SAMPLES_PER_CHANNEL_PER_ITERATION * sy.TimeSpan.MICROSECOND)
        timestamps = np.linspace(
            start, end, SAMPLES_PER_CHANNEL_PER_ITERATION, dtype=np.int64
        )
        values = np.linspace(0, 1, SAMPLES_PER_CHANNEL_PER_ITERATION, dtype=np.float32)
        self._cached_data = {
            "timestamps": np.linspace(
                start, end, SAMPLES_PER_CHANNEL_PER_ITERATION, dtype=np.int64
            ),
            **{f"channel_{i}": values for i in range(CHANNEL_COUNT)},
        }
        for i in range(ITERATIONS):
            timestamps += SAMPLES_PER_CHANNEL_PER_ITERATION * sy.TimeSpan.MICROSECOND
            if index:
                df = pd.DataFrame(self._cached_data)
                df.set_index("timestamps", inplace=True)
                df.index.name = "time"
                yield df
            else:
                df["timestamps"] = timestamps
                yield pd.DataFrame(self._cached_data)
