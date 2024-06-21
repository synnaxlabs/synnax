import sys
from typing import NamedTuple, List

import numpy as np
import synnax as sy


# Each python process opens one streamer
class TestConfig(NamedTuple):
    start_time_stamp: sy.TimeStamp
    close_after_frames: int
    channels: List[str]


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


def stream_test(tc: TestConfig):
    counter = 0
    with client.open_streamer(tc.channels, tc.start_time_stamp) as s:
        s.read()
        counter += 1
        if counter >= tc.close_after_frames:
            return


def main():
    argv = sys.argv
    start_time_stamp = int(argv[1])
    close_after_frames = int(argv[2])
    number_of_channels = int(argv[3])
    channels = []
    argv_counter = 4
    for _ in range(number_of_channels):
        channels.append(argv[argv_counter])
        argv_counter += 1

    tc = TestConfig(
        start_time_stamp=sy.TimeStamp(start_time_stamp),
        close_after_frames=close_after_frames,
        channels=channels,
    )

    stream_test(tc)


if __name__ == "__main__":
    main()
