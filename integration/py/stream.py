import sys
from typing import NamedTuple, List

import synnax as sy
from timing import time_stream


# Each python process opens one streamer
class TestConfig(NamedTuple):
    identifier: str
    start_time_stamp: sy.TimeStamp
    samples_expected: int
    channels: List[str]


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


@time_stream
def stream_test(tc: TestConfig) -> int:
    samples_streamed = 0
    with client.open_streamer(tc.channels, tc.start_time_stamp) as s:
        for f in s:
            samples_streamed += sum([len(s) for s in f.series])
            if samples_streamed >= tc.samples_expected:
                return samples_streamed


def parse_input(argv: List[str]) -> TestConfig:
    argv_counter = 1
    identifier = argv[argv_counter]
    argv_counter += 1
    start_time_stamp = int(argv[argv_counter])
    argv_counter += 1
    samples_expected = int(argv[argv_counter])
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
        samples_expected=samples_expected,
        channels=channels,
    )


def main():
    tc = parse_input(sys.argv)
    stream_test(tc)


if __name__ == "__main__":
    main()
