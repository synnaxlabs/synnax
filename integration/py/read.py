import sys
from typing import NamedTuple, List
import numpy as np

import synnax as sy
from integration import FILE_NAME


# length of channels must = num_iterators
class TestConfig(NamedTuple):
    identifier: str
    num_iterators: int
    chunk_size: int
    bounds: sy.TimeRange
    samples_expected: int
    expected_error: str
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


class Read_Test:
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
        samples_expected = int(argv[argv_counter])
        argv_counter += 1
        expected_error = argv[argv_counter]
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
            samples_expected=samples_expected,
            bounds=sy.TimeRange(sy.TimeStamp(bounds_start), sy.TimeStamp(bounds_end)),
            expected_error=expected_error,
            channels=channels,
        )

    def test_with_timing(self):
        start = sy.TimeStamp.now()
        error_assertion_passed = False
        actual_err = "no_error"
        try:
            samples = self.test()
        except Exception as e:
            actual_err = str(e)
            if (
                self._tc.expected_error != "no_error"
                and self._tc.expected_error in str(e)
            ):
                error_assertion_passed = True
            else:
                raise (e)
        else:
            if self._tc.expected_error == "no_error":
                error_assertion_passed = True
        end = sy.TimeStamp.now()

        error_assertion = f"""Expected error: {self._tc.expected_error}; Actual error: {actual_err}\n{"PASS!!" if error_assertion_passed else "FAIL!!"}"""

        s = self.generate_test_report(samples, start.span(end), error_assertion)

        with open(FILE_NAME, "a") as f:
            f.write(s)

    def generate_test_report(
        self, samples: int, time: sy.TimeSpan, error_assertion: str
    ) -> str:
        samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))
        assertion_passed = (
            "PASS!!"
            if (
                self._tc.samples_expected == 0
                or np.isclose(samples, self._tc.samples_expected, rtol=0.01)
            )
            else "FAIL!!"
        )
        assertion_result = (
            f"""Expected samples: {self._tc.samples_expected:,.2f}; Actual samples: {samples:,.2f}\n{assertion_passed}"""
            if self._tc.samples_expected != 0
            else ""
        )

        s = f"""
-- Python Read ({self._tc.identifier})--
Samples read: {samples:,.2f}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
\tNumber of iterators: {self._tc.num_iterators}
\tNumber of channels: {self._tc.num_channels()}
\tChunk size: {self._tc.chunk_size:,.0f}
{assertion_result}
{error_assertion}
"""

        return s

    def test(self) -> int:
        iterators: List[sy.Iterator] = []
        samples_read = 0

        for i in range(self._tc.num_iterators):
            iterators.append(
                client.open_iterator(
                    self._tc.bounds,
                    self._tc.channels[i],
                    self._tc.chunk_size,
                )
            )

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
    Read_Test(sys.argv).test_with_timing()


if __name__ == "__main__":
    main()
