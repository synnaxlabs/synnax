#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""EtherCAT read tasks on the USB Ethernet adapter of the HITL station. The adapter
holds some frames back past their bus cycle at 1 kHz, but not at 500 Hz."""

import re

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import StatusWatch, delete_channels, status_text
from tests.hitl.ethercat import START_TIMEOUT, STGS, create_read_task, prepare_bus

USB_WARNING = re.compile(
    r"Bus cycles failed: process data receive failed\. .+ is a USB adapter, which can "
    r"hold frames back\. Use a PCIe or onboard Ethernet adapter, or a lower sample "
    r"rate\."
)
"""The adapter name comes from the station. The Driver joins other held reasons into the
same description."""


class _Read(TestCase):
    """Runs a read task on a 6xSTG for DURATION."""

    task_name: str
    SAMPLE_RATE: sy.Rate
    DURATION: sy.TimeSpan
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


class EtherCATDroppedSamplesAt1kHz(_Read):
    """Checks that a bus that drops frames on a USB adapter warns with a text that names
    the adapter."""

    task_name = "HITL EtherCAT Dropped Samples"
    SAMPLE_RATE = 1 * sy.Rate.KHZ
    DURATION = 30 * sy.TimeSpan.SECOND

    def run(self) -> None:
        statuses, _ = self.read()
        problems = [s for s in statuses if s.variant in ("warning", "error")]
        for status in problems:
            if (status.variant, status.message) != (
                "warning",
                "Dropped samples",
            ) or not USB_WARNING.search(status.description):
                self.fail(
                    "Expected Dropped samples with the USB adapter text, but got "
                    f"{status.variant}: {status_text(status)}"
                )
        if not problems:
            self.fail(f"Expected a Dropped samples warning in {self.DURATION}")


class EtherCATSamplesAt500Hz(_Read):
    """Checks that the USB adapter keeps up at 500 Hz."""

    task_name = "HITL EtherCAT Samples"
    SAMPLE_RATE = 500 * sy.Rate.HZ
    DURATION = 3 * sy.TimeSpan.MINUTE
    MAX_MISSING = 10
    """The station measures 3 missing samples in 3 minutes."""

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
