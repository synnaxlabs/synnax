import sys
from typing import NamedTuple, List

import synnax as sy
from timing import time_stream


# Each python process opens one streamer
class TestConfig(NamedTuple):
    identifier: str
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


@time_stream("timing.log")
def stream_test(tc: TestConfig) -> int:
    counter = 0
    samples_streamed = 0
    with client.open_streamer(tc.channels, tc.start_time_stamp) as s:
        for f in s:
            counter += 1
            if counter >= tc.close_after_frames:
                samples_streamed += sum([len(s) for s in f.series])
                return samples_streamed


def parse_input(argv: List[str]) -> TestConfig:
    argv_counter = 1
    identifier = argv[argv_counter]
    argv_counter += 1
    start_time_stamp = int(argv[argv_counter])
    argv_counter += 1
    close_after_frames = int(argv[argv_counter])
    argv_counter += 1
    number_of_channels = int(argv[argv_counter])
    argv_counter += 1
    channels = []
    for _ in range(number_of_channels):
        channels.append(argv[argv_counter])
        argv_counter += 1

    return TestConfig(
        identifier = identifier,
        start_time_stamp=sy.TimeStamp(start_time_stamp),
        close_after_frames=close_after_frames,
        channels=channels,
    )


def main():
    tc = parse_input(sys.argv)
    stream_test(tc)


if __name__ == "__main__":
    main()
