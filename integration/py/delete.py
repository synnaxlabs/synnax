import sys
from typing import NamedTuple, List

import numpy as np
import synnax as sy


class IndexWriterGroup(NamedTuple):
    index_channels: List[str]
    data_channels: List[str]

    def together(self) -> List[str]:
        return self.index_channels + self.data_channels


# length of channels must = num _writers
class TestConfig(NamedTuple):
    channels: List[str]
    time_range: sy.TimeRange


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


def delete_test(tc: TestConfig):
    client.delete(tc.channels, tc.time_range)


def main():
    argv = sys.argv
    time_range_start = int(argv[1])
    time_range_end = int(argv[2])
    num_channels = int(argv[3])
    channels = []
    argv_counter = 4
    for _ in range(num_channels):
        channels.append(argv[argv_counter])
        argv_counter += 1

    tc = TestConfig(
        time_range=sy.TimeRange(sy.TimeStamp(time_range_start), sy.TimeStamp(time_range_end)),
        channels=channels,
    )

    delete_test(tc)


if __name__ == "__main__":
    main()
