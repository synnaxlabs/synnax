import sys
from typing import NamedTuple, List

import synnax as sy
from timing import time_delete

class TestConfig(NamedTuple):
    identifier: str
    channels: List[str]
    time_range: sy.TimeRange


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


@time_delete
def delete_test(tc: TestConfig):
    client.delete(tc.channels, tc.time_range)


def parse_input(argv: List[str]) -> TestConfig:
    argv_counter = 1
    identifier = argv[argv_counter]
    argv_counter += 1
    time_range_start = int(argv[argv_counter])
    argv_counter += 1
    time_range_end = int(argv[argv_counter])
    argv_counter += 1
    num_channels = int(argv[argv_counter])
    argv_counter += 1
    channels = []
    for _ in range(num_channels):
        channels.append(argv[argv_counter])
        argv_counter += 1

    return TestConfig(
        identifier=identifier,
        time_range=sy.TimeRange(sy.TimeStamp(time_range_start), sy.TimeStamp(time_range_end)),
        channels=channels,
    )


def main():
    tc = parse_input(sys.argv)
    delete_test(tc)


if __name__ == "__main__":
    main()
