import sys
from typing import NamedTuple, List

import numpy as np
import synnax as sy
from integration import FILE_NAME


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
    expected_error: str
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


class Write_Test():
    _tc: TestConfig
    
    def __init__(self, argv: List[str]):
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
        expected_error = argv[argv_counter]
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
        self._tc = TestConfig(
            identifier=identifier,
            num_writers=num_writers,
            domains=domains,
            samples_per_domain=samples_per_domain,
            time_range=sy.TimeRange(sy.TimeStamp(time_range_start), sy.TimeStamp(time_range_end)),
            channels=channel_groups,
            auto_commit=auto_commit,
            index_persist_interval=index_persist_interval,
            expected_error=expected_error,
            writer_mode=writer_mode,
        )

        
    def test_with_timing(self):
        start = sy.TimeStamp.now()
        error_assertion_passed = False
        actual_error = "no_error"
        try:
            self.test()
        except Exception as e:
            actual_error = str(e)
            if self._tc.expected_error != "no_error" and self._tc.expected_error in str(e):
                error_assertion_passed = True
        else:
            actual_error = "no_error"
            if self._tc.expected_error == "no_error":
                error_assertion_passed = True
        end = sy.TimeStamp.now()

        err_assertion = f'''
Expected error: {self._tc.expected_error}; Actual error: {actual_error}\n{"PASS!!" if error_assertion_passed else "FAIL!!"}
'''

        s = self.generate_test_report(start.span(end), err_assertion)
        
        with open(FILE_NAME, "a") as f:
            f.write(s)

            
    def generate_test_report(self, time: sy.TimeSpan, err_assertion: str) -> str:
        samples = self._tc.num_channels() * self._tc.samples_per_domain * self._tc.domains
        samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))
        s = f'''
-- Python Write ({self._tc.identifier}) --
Samples written: {samples}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
\tNumber of writers: {self._tc.num_writers}
\tNumber of channels: {self._tc.num_channels()}
\tNumber of domains: {self._tc.domains:,.0f}
\tSamples per domain: {self._tc.samples_per_domain:,.0f}
\tAuto commit: {str(self._tc.auto_commit)}
\tIndex persist interval: {self._tc.index_persist_interval}
\tWriter mode: {sy.WriterMode(self._tc.writer_mode).name}
{err_assertion}

'''

        return s

    def test(self):
        writers = [None] * self._tc.num_writers
        time_span_per_domain = self._tc.time_range.span / self._tc.domains

        for i in range(self._tc.num_writers):
            writers[i] = client.open_writer(
                start=self._tc.time_range.start,
                channels=self._tc.channels[i].together(),
                name=f"writer{i}",
                mode=self._tc.writer_mode,
                enable_auto_commit=self._tc.auto_commit,
                auto_index_persist_interval=self._tc.index_persist_interval,
            )

        try:
            ts_hwm = self._tc.time_range.start
            for _ in range(self._tc.domains):
                timestamps = np.linspace(
                    ts_hwm,
                    ts_hwm + time_span_per_domain,
                    self._tc.samples_per_domain,
                    dtype="int64",
                    )
                data = np.sin(0.0000000001 * timestamps)
                for i, writer in enumerate(writers):
                    data_dict = {}
                    for index_channel in self._tc.channels[i].index_channels:
                        data_dict[index_channel] = timestamps
                    for data_channel in self._tc.channels[i].data_channels:
                        data_dict[data_channel] = data

                    writer.write(data_dict)

                    if not self._tc.auto_commit:
                        assert writer.commit()

                ts_hwm += time_span_per_domain + 1

        finally:
            for writer in writers:
                writer.close()


def main():
    Write_Test(sys.argv).test_with_timing()


if __name__ == "__main__":
    main()
