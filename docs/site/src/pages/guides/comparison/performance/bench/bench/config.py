import numpy as np
import pandas as pd
from typing import Iterator
import synnax as sy

SAMPLES_PER_CHANNEL = 100_000
SAMPLES_PER_CHANNEL_PER_ITERATION = 1000
ITERATIONS = SAMPLES_PER_CHANNEL // SAMPLES_PER_CHANNEL_PER_ITERATION
CHANNEL_COUNT = 20
TOTAL_SAMPLES = SAMPLES_PER_CHANNEL * ITERATIONS * CHANNEL_COUNT

START_TIME = sy.TimeStamp.now()

class TestConfig:
    _channels: list[sy.Channel]
    _start_time = sy.TimeStamp.now()
    _cached_data: pd.DataFrame | None = None
    channel_count = CHANNEL_COUNT
    samples_per_channel = SAMPLES_PER_CHANNEL
    samples_per_channel_per_iteration = SAMPLES_PER_CHANNEL_PER_ITERATION

    def __init__(
        self,
        channel_count: int = CHANNEL_COUNT,
        samples_per_channel: int = SAMPLES_PER_CHANNEL,
        samples_per_channel_per_iteration: int = SAMPLES_PER_CHANNEL_PER_ITERATION
    ):
        self.channel_count = channel_count
        self.samples_per_channel = samples_per_channel
        self.samples_per_channel_per_iteration = samples_per_channel_per_iteration
        c = channel_count
        c -= 1
        idx = sy.Channel(
            name="timestamps",
            data_type="timestamp",
            is_index=True
        )
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

    @property
    def iterations(self) -> int:
        return self.samples_per_channel // self.samples_per_channel_per_iteration

    def frames(self, index: bool = False) -> Iterator[pd.DataFrame]:
        start = self._start_time
        end = start + (self.samples_per_channel_per_iteration * sy.TimeSpan.MICROSECOND)
        timestamps = np.linspace(start, end, self.samples_per_channel_per_iteration, dtype=np.int64)
        values = np.linspace(0, 1, self.samples_per_channel_per_iteration, dtype=np.float32)
        self._cached_data = {
            "timestamps": np.linspace(start, end, self.samples_per_channel_per_iteration, dtype=np.int64),
            **{f"channel_{i}": values for i in range(self.channel_count - 1)}
        }
        for i in range(self.iterations):
            timestamps += (self.samples_per_channel_per_iteration * sy.TimeSpan.MICROSECOND)
            if index:
                df = pd.DataFrame(self._cached_data)
                df.set_index('timestamps', inplace=True)
                df.index.name = 'timestamps'
                yield df
            else:
                df = pd.DataFrame(self._cached_data)
                df['timestamps'] = timestamps
                yield df

    def __str__(self) -> str:
        return f"TestConfig({self.channel_count=}, {self.samples_per_channel=}, {self.samples_per_channel_per_iteration=})"
