import sys
from typing import NamedTuple, List

import synnax as sy
from integration import FILE_NAME

class TestConfig(NamedTuple):
    identifier: str
    expected_error: str
    channels: List[str]
    time_range: sy.TimeRange


client = sy.Synnax(
    host="localhost",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)


class Delete_Test:
    _tc: TestConfig
    def __init__(self, argv: List[str]):
        argv_counter = 1
        identifier = argv[argv_counter]
        argv_counter += 1
        time_range_start = int(argv[argv_counter])
        argv_counter += 1
        time_range_end = int(argv[argv_counter])
        argv_counter += 1
        expected_error = argv[argv_counter]
        argv_counter += 1
        num_channels = int(argv[argv_counter])
        argv_counter += 1
        channels = []
        for _ in range(num_channels):
            channels.append(argv[argv_counter])
            argv_counter += 1

        self._tc = TestConfig(
            identifier=identifier,
            time_range=sy.TimeRange(sy.TimeStamp(time_range_start), sy.TimeStamp(time_range_end)),
            expected_error=expected_error,
            channels=channels,
        )


    def test(self):
        client.delete(self._tc.channels, self._tc.time_range)


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
                raise(e)
        else:
            actual_error = "no_error"
            if self._tc.expected_error == "no_error":
                error_assertion_passed = True
        end = sy.TimeStamp.now()

        s = f'''
-- Python Delete ({self._tc.identifier})--
Time taken: {start.span(end)}
Configuration:
\tNumber of channels: {len(self._tc.channels)}
Expected error: {self._tc.expected_error}; Actual error: {actual_error}\n{"PASS!!" if error_assertion_passed else "FAIL!!!!"}
'''
        with open(FILE_NAME, "a") as f:
            f.write(s)


def main():
    Delete_Test(sys.argv).test_with_timing()


if __name__ == "__main__":
    main()
