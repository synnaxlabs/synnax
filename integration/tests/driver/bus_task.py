#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Cases of the bus integrations, which ship dark."""

from abc import abstractmethod
from typing import Unpack

import synnax as sy
from framework import drivers
from framework.test_case import Config
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    ReadTask,
    ReadTaskCase,
    Task,
    WriteTaskCase,
    create_channel,
    create_index,
)

INTEGRATIONS = ["serial", "tcp", "udp"]
"""The bus integrations, which the Driver of a run leaves off."""
DRIVER_UP_TIMEOUT = 30 * sy.TimeSpan.SECOND


class BusCase(SimulatorCase):
    """Base class for cases of the bus integrations. The Driver of the run leaves them
    off, so each case runs a standalone Driver of its own that runs only them. The
    devices of the simulators sit on the rack of that Driver."""

    def __init__(self, **kwargs: Unpack[Config]) -> None:
        super().__init__(**kwargs)
        self.driver: drivers.Process | None = None
        self._rack: sy.Rack | None = None

    def setup(self) -> None:
        """Start the Driver and wait until it runs, then delegate to next in MRO."""
        binary = self.options.driver_binary
        if binary is None:
            self.skip("needs --driver-binary")
        self.driver = drivers.Process(
            binary=binary,
            connection=self.synnax_connection,
            log_path=self.results_dir / "driver.log",
            integrations=INTEGRATIONS,
        )
        self._rack = self._wait_for_rack(self.driver)
        super().setup()

    def device_rack(self) -> sy.Rack:
        assert self._rack is not None, "setup() must run before device_rack()"
        return self._rack

    def _wait_for_rack(self, driver: drivers.Process) -> sy.Rack:
        """Return the rack of driver once the Driver reports itself running."""
        api_key = sy.poll(driver.api_key, timeout=DRIVER_UP_TIMEOUT, interval=0.2)
        if api_key is None:
            self.fail(f"the Driver did not enroll in {DRIVER_UP_TIMEOUT}")
        key = self._rack_key(api_key)
        self.track_test_racks([self.client.racks.retrieve(key)])

        def running() -> sy.Rack | None:
            rack = self.client.racks.retrieve(key, include_status=True)
            status = rack.status
            if status is None or status.details is None or not status.details.running:
                return None
            return rack

        rack = sy.poll(running, timeout=DRIVER_UP_TIMEOUT, interval=0.2)
        if rack is None:
            self.fail(
                f"the Driver did not report itself running in {DRIVER_UP_TIMEOUT}"
            )
        return rack

    def _rack_key(self, api_key: str) -> int:
        """Return the key of the rack that api_key logs in as."""
        c = self.synnax_connection
        client = sy.Synnax(
            host=c.server_address,
            port=c.port,
            secure=c.secure,
            ca_cert=c.ca_cert,
            api_key=api_key,
        )
        try:
            subject = client.auth.subject
        finally:
            client.close()
        if subject is None or subject.type != "rack" or subject.key is None:
            self.fail(f"the API key of the Driver logs in as {subject}, not a rack")
        return int(subject.key)

    def teardown(self) -> None:
        """Stop the Driver, then run the remaining teardown, which deletes its rack."""
        try:
            if self.driver is not None:
                with self._try_to("stop the Driver"):
                    self.driver.stop()
        finally:
            super().teardown()


class BusReadCase[ReadTaskT: ReadTask](BusCase, ReadTaskCase[ReadTaskT]):
    """Base class for cases that read the line of reading that a bus simulator sends at
    SAMPLE_RATE: a temperature and a pressure, such as "21.500,101.300", stamped when
    the line arrives."""

    EVENT_DRIVEN = True
    SAMPLE_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 3 * sy.TimeSpan.SECOND
    channel_prefix: str
    """The start of the names of the channels of the case."""

    def __init__(self, **kwargs: Unpack[Config]) -> None:
        super().__init__(**kwargs)
        self._channels: list[int] = []

    @abstractmethod
    def task(
        self, *, name: str, device: sy.device.Key, entries: list[sy.icd.ReadEntry]
    ) -> ReadTaskT:
        """Return the read task named name that reads entries from device."""

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> ReadTaskT:
        idx = create_index(self.client, f"{self.channel_prefix}_time")
        self._channels = [
            create_channel(
                self.client,
                name=f"{self.channel_prefix}_{name}",
                data_type=sy.DataType.FLOAT64,
                index=idx.key,
            )
            for name in ("temperature", "pressure")
        ]
        fields: list[sy.icd.TextField] = [
            sy.icd.DelimitedTextField(name="temperature", position=0),
            sy.icd.DelimitedTextField(name="pressure", position=1),
        ]
        message = sy.icd.Message(
            name="reading", payload=sy.icd.TextPayload(fields=fields)
        )
        ic = self.client.icds.create(
            sy.icd.ICD(
                name=device.name,
                messages=[message],
                framing=sy.icd.DelimitedFraming(),
            )
        )
        self.track_test_icds([ic])
        bindings = [
            sy.icd.ReadBinding(field=f.key, channel=ch)
            for f, ch in zip(fields, self._channels, strict=True)
        ]
        return self.task(
            name=task_name,
            device=device.key,
            entries=[
                sy.icd.ReadEntry(
                    icd=ic.key,
                    messages=[
                        sy.icd.ReadMessage(message=message.key, bindings=bindings)
                    ],
                )
            ],
        )

    def _channel_keys(self, task: ReadTaskT) -> list[int]:
        return self._channels


class BusWriteCase[TaskT: Task](BusCase, WriteTaskCase[TaskT]):
    """Base class for cases that send a setpoint line to a bus simulator for each
    command, such as "SET,37.25"."""

    command_values: list[list[float]] | None = [[12.5], [37.25]]
    # The lines of command_values, as the Driver writes the shortest decimal text.
    LINES = ["SET,12.5", "SET,37.25"]
    channel_prefix: str
    """The start of the names of the channels of the case."""

    def __init__(self, **kwargs: Unpack[Config]) -> None:
        super().__init__(**kwargs)
        self._channels: list[int] = []
        self._checked = 0

    @abstractmethod
    def task(
        self, *, name: str, device: sy.device.Key, targets: list[sy.icd.WriteTarget]
    ) -> TaskT:
        """Return the write task named name that sends targets to device."""

    @abstractmethod
    def received(self) -> list[str]:
        """Return the lines that the simulator has received, oldest first."""

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> TaskT:
        idx = create_index(self.client, f"{self.channel_prefix}_cmd_time")
        cmd = create_channel(
            self.client,
            name=f"{self.channel_prefix}_setpoint_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        self._channels = [cmd]
        field = sy.icd.DelimitedTextField(name="setpoint", position=1)
        message = sy.icd.Message(
            name="setpoint",
            payload=sy.icd.TextPayload(prefix="SET", fields=[field]),
        )
        ic = self.client.icds.create(
            sy.icd.ICD(
                name=device.name,
                messages=[message],
                framing=sy.icd.DelimitedFraming(),
            )
        )
        self.track_test_icds([ic])
        return self.task(
            name=task_name,
            device=device.key,
            targets=[
                sy.icd.WriteTarget(
                    icd=ic.key,
                    message=sy.icd.WriteMessage(
                        message=message.key,
                        bindings=[
                            sy.icd.ChannelWriteBinding(field=field.key, channel=cmd)
                        ],
                    ),
                )
            ],
        )

    def _channel_keys(self, task: TaskT) -> list[int]:
        return self._channels

    def assert_written(self) -> None:
        """Check that the lines since the last check end with LINES. A retried round
        sends its lines again before them."""

        def written() -> list[str] | None:
            lines = self.received()[self._checked :]
            return lines if lines[-len(self.LINES) :] == self.LINES else None

        lines = sy.poll(written, timeout=self.COMMAND_TIMEOUT, interval=0.1)
        if lines is None:
            new = self.received()[self._checked :]
            self.fail(f"the simulator got {new}, not {self.LINES} at the end")
        self._checked += len(lines)
