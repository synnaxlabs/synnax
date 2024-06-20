import multiprocessing
import random
import time
from typing import NamedTuple, List

import numpy as np
import synnax as sy


class TestConfig(NamedTuple):
    channels: List[str]
    commit_frequency: sy.TimeSpan
    samples_per_commit: int


tc = TestConfig(10, 490, 10, 10 * sy.TimeSpan.SECOND, sy.TimeSpan.MILLISECOND, 1000)

client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)

frames_read = 0

with client.open_streamer(tc.channels) as s:
    for frame in s:
        frames_read += 1
        frame.len
