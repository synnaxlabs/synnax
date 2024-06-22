import sys
from typing import NamedTuple, List

import synnax as sy
from timing import time_read


# length of channels must = num_iterators
class TestConfig(NamedTuple):
    num_iterators: int
    chunk_size: int
    bounds: sy.TimeRange
    channels: List[List[str]]

    def num_channels(self):
        return sum([len(ch) for ch in self.channels])


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


@time_read("timing.log")
def read_test(tc: TestConfig) -> int:
    iterators: List[sy.Iterator] = []
    samples_read = 0

    for i in range(tc.num_iterators):
        iterators.append(client.open_iterator(tc.bounds, tc.channels[i], tc.chunk_size))

    try:
        for i in iterators:
            i.seek_first()
            while i.next(sy.AUTO_SPAN):
                samples_read += sum([len(s) for s in i.value.series])
                continue
    finally:
        for iterator in iterators:
            iterator.close()

    return samples_read


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
