#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""EtherCAT read tasks on the USB Ethernet adapter of the HITL station. The adapter
holds some frames back past their bus cycle at 1 kHz, and the Driver takes them late."""

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import StatusWatch, delete_channels, status_text
from tests.hitl.ethercat import START_TIMEOUT, STGS, create_read_task, prepare_bus


class _Read(TestCase):
    """Runs a read task on a 6xSTG for DURATION, and checks that it shows no warning and
    misses fewer than MAX_MISSING samples."""

    task_name: str
    SAMPLE_RATE: sy.Rate
    DURATION: sy.TimeSpan
    MAX_MISSING: int
    tsk: sy.ethercat.ReadTask | None = None

    def setup(self) -> None:
        slaves = prepare_bus(self.client)
        self.tsk = create_read_task(
            self.client,
            [slaves[STGS[0]]],
            name=self.task_name,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=25 * sy.Rate.HZ,
        )
        self.client.tasks.configure(self.tsk)

    def teardown(self) -> None:
        if self.tsk is not None:
            with self._try_to(f"delete task {self.tsk.name}"):
                self.client.tasks.delete(self.tsk.key)
            with self._try_to("delete the task channels"):
                delete_channels(
                    self.client, [ch.channel for ch in self.tsk.config.channels]
                )
        super().teardown()

    def read(self) -> tuple[list[sy.task.Status], int]:
        """Returns the statuses of the task while it runs, and the number of samples
        that it misses."""
        assert self.tsk is not None
        with StatusWatch(self.client, [self.tsk.key]) as watch:
            with self.tsk.run(timeout=START_TIMEOUT):
                start = sy.TimeStamp.now()
                watch.collect(self.DURATION)
                end = sy.TimeStamp.now()
        first = self.client.channels.retrieve(self.tsk.config.channels[0].channel)
        index = self.client.channels.retrieve(first.index)
        times = index.read(sy.TimeRange(start, end)).to_numpy()
        if len(times) < 2:
            self.fail(f"Expected task {self.tsk.name} to read data")
        period = int(self.SAMPLE_RATE.period)
        expected = round(int(times[-1] - times[0]) / period) + 1
        self.log(f"Read {len(times)} of {expected} samples")
        return watch.seen, expected - len(times)

    def run(self) -> None:
        statuses, missing = self.read()
        for status in statuses:
            if status.variant in ("warning", "error"):
                self.fail(f"Expected no warning, but got {status_text(status)}")
        if missing >= self.MAX_MISSING:
            self.fail(
                f"Expected fewer than {self.MAX_MISSING} missing samples, but missed "
                f"{missing}"
            )


class EtherCATSamplesAt1kHz(_Read):
    """Checks that the Driver takes the frames that the USB adapter holds back at
    1 kHz."""

    task_name = "HITL EtherCAT Samples at 1 kHz"
    SAMPLE_RATE = 1 * sy.Rate.KHZ
    DURATION = 30 * sy.TimeSpan.SECOND
    MAX_MISSING = 20
    """The station measures 1 to 3 missing samples in 30 s, and about 200 when the
    Driver drops each late frame."""


class EtherCATSamplesAt500Hz(_Read):
    """Checks that the USB adapter keeps up at 500 Hz."""

    task_name = "HITL EtherCAT Samples"
    SAMPLE_RATE = 500 * sy.Rate.HZ
    DURATION = 3 * sy.TimeSpan.MINUTE
    MAX_MISSING = 10
    """The station measures 3 missing samples in 3 minutes."""
