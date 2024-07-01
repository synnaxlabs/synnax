import sys
from typing import NamedTuple, List

import synnax as sy
from timing import time_read
FILE_NAME = "../timing.log"


# length of channels must = num_iterators
class TestConfig(NamedTuple):
    identifier: str
    num_iterators: int
    chunk_size: int
    bounds: sy.TimeRange
    expected_samples: int
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

class Read_Test():
    _tc: TestConfig

    def __init__(self, argv: List[str]):
        argv_counter = 1
        identifier = argv[argv_counter]
        argv_counter += 1
        num_iterators = int(argv[argv_counter])
        argv_counter += 1
        chunk_size = int(argv[argv_counter])
        argv_counter += 1
        bounds_start = int(argv[argv_counter])
        argv_counter += 1
        bounds_end = int(argv[argv_counter])
        argv_counter += 1
        expected_samples = int(argv[argv_counter])
        argv_counter += 1
        number_of_channel_groups = int(argv[argv_counter])
        argv_counter += 1
        channels = []
        for _ in range(number_of_channel_groups):
            number_of_channels_in_group = int(argv[argv_counter])
            argv_counter += 1
            channel_group = []
            for _ in range(number_of_channels_in_group):
                channel_group.append(argv[argv_counter])
                argv_counter += 1
            channels.append(channel_group)

        self._tc = TestConfig(
            identifier=identifier,
            num_iterators=num_iterators,
            chunk_size=chunk_size,
            expected_samples=expected_samples,
            bounds=sy.TimeRange(sy.TimeStamp(bounds_start), sy.TimeStamp(bounds_end)),
            channels=channels,
        )


    def testWithTiming(self):
        start = sy.TimeStamp.now()
        samples = self.test()
        end = sy.TimeStamp.now()

        time: sy.TimeSpan = start.span(end)
        samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))
        assertion_passed = self._tc.expected_samples == samples
        assertion_result = f'''
\tExpected samples: {self._tc.expected_samples}; Actual samples: {samples}
''' if self._tc.expected_samples != 0 else ""
        
        s = f'''
-- Python Read ({self._tc.identifier})--
Samples read: {samples}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
\tNumber of iterators: {self._tc.num_iterators}
\tNumber of channels: {self._tc.num_channels()}
\tChunk size: {self._tc.chunk_size:,.0f}
{assertion_result}
        '''
        with open(FILE_NAME, "a") as f:
            f.write(s)


    def test(self) -> int:
        iterators: List[sy.Iterator] = []
        samples_read = 0

        for i in range(self._tc.num_iterators):
            iterators.append(client.open_iterator(
                self._tc.bounds,
                self._tc.channels[i],
                self._tc.chunk_size,
            ))

        try:
            for i in iterators:
                i.seek_first()
                while i.next(sy.AUTO_SPAN):
                    samples_read += sum([len(s) for s in i.value.series])
        finally:
            for iterator in iterators:
                iterator.close()

        return samples_read


def main():
    Read_Test(sys.argv).testWithTiming()


if __name__ == "__main__":
    main()
