import sys
from typing import NamedTuple, List

import numpy as np
import synnax as sy
from timing import time_write


class IndexWriterGroup(NamedTuple):
    index_channels: List[str]
    data_channels: List[str]

    def together(self) -> List[str]:
        return self.index_channels + self.data_channels

    def __len__(self) -> int:
        return len(self.index_channels) + len(self.data_channels)


# length of channels must = num _writers
class TestConfig(NamedTuple):
    identifier: str
    num_writers: int
    domains: int
    samples_per_domain: int
    time_range: sy.TimeRange
    auto_commit: bool
    index_persist_interval: sy.TimeSpan
    writer_mode: sy.WriterMode
    channels: List[IndexWriterGroup]

    def num_channels(self) -> int:
        return sum([len(channelGroup) for channelGroup in self.channels])


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


@time_write("timing.log")
def write_test(tc: TestConfig):
    writers = [None] * tc.num_writers
    time_span_per_domain = tc.time_range.span / tc.domains

    for i in range(tc.num_writers):
        writers[i] = client.open_writer(
            start=tc.time_range.start,
            channels=tc.channels[i].together(),
            name=f"writer{i}",
            mode=tc.writer_mode,
            enable_auto_commit=tc.auto_commit,
            auto_index_persist_interval=tc.index_persist_interval,
        )

    try:
        ts_hwm = tc.time_range.start
        for _ in range(tc.domains):
            timestamps = np.linspace(
                ts_hwm,
                ts_hwm + time_span_per_domain,
                tc.samples_per_domain,
                dtype="int64",
                )
            data = np.sin(0.0000000001 * timestamps)
            for i, writer in enumerate(writers):
                data_dict = {}
                for index_channel in tc.channels[i].index_channels:
                    data_dict[index_channel] = timestamps
                for data_channel in tc.channels[i].data_channels:
                    data_dict[data_channel] = data

                writer.write(data_dict)

                if not tc.auto_commit:
                    assert writer.commit()

            ts_hwm += time_span_per_domain + 1

    finally:
        for writer in writers:
            writer.close()


def parse_input(argv: List[str]) -> TestConfig:
    argv_counter = 1
    identifier = argv[argv_counter]
    argv_counter += 1
    num_writers = int(argv[argv_counter])
    argv_counter += 1
    domains = int(argv[argv_counter])
    argv_counter += 1
    samples_per_domain = int(argv[argv_counter])
    argv_counter += 1
    time_range_start = int(argv[argv_counter])
    argv_counter += 1
    time_range_end = int(argv[argv_counter])
    argv_counter += 1
    auto_commit = argv[argv_counter] == "true"
    argv_counter += 1
    index_persist_interval = sy.TimeSpan(int(argv[argv_counter]))
    argv_counter += 1
    writer_mode = sy.WriterMode(int(argv[argv_counter]))
    argv_counter += 1
    number_of_channel_groups = int(argv[argv_counter])
    argv_counter += 1
    channel_groups = []
    for _ in range(number_of_channel_groups):
        number_of_index = int(argv[argv_counter])
        index_channels = []
        argv_counter += 1
        number_of_data = int(argv[argv_counter])
        data_channels = []
        argv_counter += 1
        for i in range(argv_counter, argv_counter + number_of_index):
            index_channels.append(argv[i])
        argv_counter += number_of_index
        for i in range(argv_counter, argv_counter + number_of_data):
            data_channels.append(argv[i])
        argv_counter += number_of_data
        channel_groups.append(
            IndexWriterGroup(index_channels=index_channels, data_channels=data_channels)
        )
    return TestConfig(
        identifier=identifier,
        num_writers=num_writers,
        domains=domains,
        samples_per_domain=samples_per_domain,
        time_range=sy.TimeRange(sy.TimeStamp(time_range_start), sy.TimeStamp(time_range_end)),
        channels=channel_groups,
        auto_commit=auto_commit,
        index_persist_interval=index_persist_interval,
        writer_mode=writer_mode,
    )


def main():
    tc = parse_input(sys.argv)
    write_test(tc)


if __name__ == "__main__":
    main()
