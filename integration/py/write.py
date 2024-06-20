import multiprocessing
import random
import time
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
    num_writers: int
    domains: int
    samples_per_domain: int
    channels: List[IndexWriterGroup]


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


def write_test(tc: TestConfig):
    writers = [None] * tc.num_writers

    for i in range(tc.num_writers):
        writers[i] = client.open_writer(
            start=sy.TimeStamp.now(),
            channels=tc.channels[i].together(),
            name=f"writer{i}",
            mode=sy.WriterMode.PERSIST_STREAM,
            enable_auto_commit=False,
        )

    try:
        ts_hwm = sy.TimeStamp.now()
        for _ in range(tc.domains):
            timestamps = np.linspace(
                ts_hwm,
                ts_hwm + sy.TimeSpan.SECOND,
                tc.samples_per_domain,
                dtype="int64",
            )
            data = np.sin(timestamps) * 100
            for i, writer in enumerate(writers):
                data_dict = {}
                for index_channel in tc.channels[i].index_channels:
                    data_dict[index_channel] = timestamps
                for data_channel in tc.channels[i].data_channels:
                    data_dict[data_channel] = data

                writer.write(data_dict)
                assert writer.commit()

            ts_hwm += sy.TimeSpan.SECOND + 1

    finally:
        for writer in writers:
            writer.close()


def main():
    argv = sys.argv
    num_writers = int(argv[1])
    domains = int(argv[2])
    samples_per_domain = int(argv[3])
    number_of_channel_groups = int(argv[4])
    argv_counter = 5
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
    tc = TestConfig(
        num_writers=num_writers,
        domains=domains,
        samples_per_domain=samples_per_domain,
        channels=channel_groups,
    )

    write_test(tc)


if __name__ == "__main__":
    main()
