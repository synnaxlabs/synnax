import sys
from typing import NamedTuple, List

import synnax as sy


# length of channels must = num_iterators
class TestConfig(NamedTuple):
    num_iterators: int
    chunk_size: int
    bounds: sy.TimeRange
    channels: List[List[str]]


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


def read_test(tc: TestConfig):
    iterators = [None] * tc.num_iterators

    for i in range(tc.num_iterators):
        iterators[i] = client.open_iterator(tc.bounds, tc.channels[i], tc.chunk_size)
        iterators[i].seek_first(tc.bounds.start)

    try:
        for i in iterators:
            while i.next(sy.AUTO_SPAN):
                continue
    finally:
        for iterator in range(iterators):
            iterator.close()


def main():
    argv = sys.argv
    num_iterators = int(argv[1])
    chunk_size = int(argv[2])
    bounds_start = int(argv[3])
    bounds_end = int(argv[4])
    number_of_channel_groups = int(argv[5])
    argv_counter = 6
    channels = []
    for _ in range(number_of_channel_groups):
        number_of_channels_in_group = int(argv[argv_counter])
        argv_counter += 1
        channel_group = []
        for _ in range(number_of_channels_in_group):
            channel_group.append(argv[argv_counter])
            argv_counter += 1
        channels.append(channel_group)

    tc = TestConfig(
        num_iterators=num_iterators,
        chunk_size=chunk_size,
        bounds=sy.TimeRange(sy.TimeStamp(bounds_start), sy.TimeStamp(bounds_end)),
        channels=channels,
    )

    read_test(tc)


if __name__ == "__main__":
    main()
