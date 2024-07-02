import sys
from typing import NamedTuple, List

import synnax as sy
from integration import FILE_NAME

# Each python process opens one streamer
class TestConfig(NamedTuple):
    identifier: str
    start_time_stamp: sy.TimeStamp
    samples_expected: int
    expected_error: str
    channels: List[str]


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


class Stream_Test():
    _tc: TestConfig
    
    def __init__(self, argv: List[str]):
        argv_counter = 1
        identifier = argv[argv_counter]
        argv_counter += 1
        start_time_stamp = int(argv[argv_counter])
        argv_counter += 1
        samples_expected = int(argv[argv_counter])
        argv_counter += 1
        expected_error = argv[argv_counter]
        argv_counter += 1
        number_of_channels = int(argv[argv_counter])
        argv_counter += 1
        channels = []
        for _ in range(number_of_channels):
            channels.append(argv[argv_counter])
            argv_counter += 1

        self._tc = TestConfig(
            identifier = identifier,
            start_time_stamp=sy.TimeStamp(start_time_stamp),
            samples_expected=samples_expected,
            expected_error=expected_error,
            channels=channels,
        )

    def test_with_timing(self):
        start = sy.TimeStamp.now()
        error_assertion_passed = False
        actual_error = "no_error"
        samples = 0
        try:
            samples = self.test()
        except Exception as e:
            actual_error = str(e)
            if self._tc.expected_error != "no_error" and self._tc.expected_error in str(e):
                error_assertion_passed = True
        else:
            actual_error = "no_error"
            if self._tc.expected_error == "no_error":
                error_assertion_passed = True
        end = sy.TimeStamp.now()
        time = start.span(end)
        samples_per_second = samples / (float(time) / float(sy.TimeSpan.SECOND))

        err_assertion = f'''
Expected error: {self._tc.expected_error}; Actual error: {actual_error}\n{"PASS!!" if error_assertion_passed else "FAIL!!"}
'''
        s = f'''
-- Python Stream ({self._tc.identifier})--
Samples streamed: {samples}
Time taken: {time}
Calculated Samples per Second: {samples_per_second:,.2f}
Configuration:
\tNumber of streamers: 1
\tNumber of channels: {len(self._tc.channels)}
\tSamples expected: {self._tc.samples_expected}
{err_assertion}

            '''
        with open(FILE_NAME, "a") as f:
            f.write(s)


    def test(self) -> int:
        samples_streamed = 0
        with client.open_streamer(self._tc.channels, self._tc.start_time_stamp) as s:
            for f in s:
                samples_streamed += sum([len(s) for s in f.series])
                if samples_streamed >= self._tc.samples_expected:
                    return samples_streamed





def main():
    Stream_Test(sys.argv).test_with_timing()


if __name__ == "__main__":
    main()
