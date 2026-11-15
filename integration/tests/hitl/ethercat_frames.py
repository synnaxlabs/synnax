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

from pydantic import ValidationError

import synnax as sy
from framework.test_case import TestCase
from tests.driver.task import delete_channels, status_text
from tests.hitl.ethercat import START_TIMEOUT, STGS, create_read_task, prepare_bus

DURATION = 3 * sy.TimeSpan.MINUTE
USB_WARNING = re.compile(
    r"Bus cycles failed: process data receive failed\. .+ is a USB adapter, which can "
    r"hold frames back\. Use a PCIe or onboard Ethernet adapter, or lower the rate of "
    r"the fastest task on the network\."
)
"""The adapter name comes from the station."""
HOLD = 10 * sy.TimeSpan.SECOND
"""How long the Driver keeps the warning after the last dropped sample."""
MIN_GAP = HOLD - 1100 * sy.TimeSpan.MILLISECOND
"""The Driver can send a status up to one 1 s window late, and stamps it with its wall
clock, so a held warning can clear a little under 9 s after the last warning."""

_STATUS_CHANNEL = "sy_status_set"
_INDEX = "hitl_ethercat_time"


class _Read(TestCase):
    """Runs a read task on a 6xSTG for DURATION."""

    task_name: str
    SAMPLE_RATE: sy.Rate
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
        key = self.tsk.key
        statuses: list[sy.task.Status] = []
        with self.client.open_streamer([_STATUS_CHANNEL]) as streamer:
            with self.tsk.run(timeout=START_TIMEOUT):
                start = sy.TimeStamp.now()
                timer = sy.Timer()
                while timer.elapsed() < DURATION:
                    frame = streamer.read(timeout=1)
                    if frame is None or _STATUS_CHANNEL not in frame:
                        continue
                    for raw in frame[_STATUS_CHANNEL]:
                        try:
                            status = sy.task.Status.model_validate(raw)
                        except ValidationError:
                            continue
                        if status.owner == sy.task.resource_id(key):
                            statuses.append(status)
                end = sy.TimeStamp.now()
        index = self.client.channels.retrieve(_INDEX)
        times = index.read(sy.TimeRange(start, end)).to_numpy()
        if len(times) < 2:
            self.fail(f"Expected task {self.tsk.name} to read data")
        period = int(self.SAMPLE_RATE.period)
        expected = round(int(times[-1] - times[0]) / period) + 1
        self.log(f"Read {len(times)} of {expected} samples")
        return statuses, expected - len(times)


class EtherCATDroppedSamplesAt1kHz(_Read):
    """Checks that a bus that drops frames on a USB adapter warns with a text that names
    the adapter, and keeps each warning until the bus is quiet for HOLD."""

    task_name = "HITL EtherCAT Dropped Samples"
    SAMPLE_RATE = 1 * sy.Rate.KHZ

    def run(self) -> None:
        statuses, _ = self.read()
        last_warning: sy.TimeStamp | None = None
        for status in statuses:
            if status.variant == "warning":
                if status.message != "Dropped samples" or not USB_WARNING.fullmatch(
                    status.description
                ):
                    self.fail(
                        "Expected Dropped samples with the USB adapter text, but got "
                        f"{status_text(status)}"
                    )
                self.log(f"{status.time}: Dropped samples")
                last_warning = status.time
            elif last_warning is None:
                continue
            elif status.variant == "success" and status.message == "Task running":
                gap = status.time - last_warning
                self.log(f"{status.time}: Task running, {gap} after the last warning")
                if gap < MIN_GAP:
                    self.fail(
                        f"Expected the warning to hold for at least {MIN_GAP}, but "
                        f"Task running came {gap} after it"
                    )
            else:
                self.fail(
                    "Expected only the warning and Task running, but got "
                    f"{status.variant}: {status_text(status)}"
                )
        if last_warning is None:
            self.fail(f"Expected a Dropped samples warning in {DURATION}")


class EtherCATSamplesAt500Hz(_Read):
    """Checks that the USB adapter keeps up at 500 Hz."""

    task_name = "HITL EtherCAT Samples"
    SAMPLE_RATE = 500 * sy.Rate.HZ
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
