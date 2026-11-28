#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
from typing import Unpack

from examples.serial_sim import SerialSim

import synnax as sy
from framework.test_case import Config
from synnax import serial
from tests.driver.bus_task import BusCase
from tests.driver.task import WriteTaskCase, create_channel, create_index


class SerialWrite(BusCase, WriteTaskCase[serial.WriteTask]):
    """Sends a setpoint line to the serial simulator for each command, such as
    "SET,37.25"."""

    sim_classes = [SerialSim]
    task_name = "Serial Write"
    command_values: list[list[float]] | None = [[12.5], [37.25]]
    # The lines of command_values, as the Driver writes the shortest decimal text.
    LINES = ["SET,12.5", "SET,37.25"]

    def __init__(self, **kwargs: Unpack[Config]) -> None:
        super().__init__(**kwargs)
        self._checked = 0

    def setup(self) -> None:
        if sys.platform == "win32":
            self.skip("Pseudo-terminals need POSIX")
        super().setup()

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> serial.WriteTask:
        idx = create_index(self.client, "serial_setpoint_cmd_time")
        cmd = create_channel(
            self.client,
            name="serial_setpoint_cmd",
            data_type=sy.DataType.FLOAT64,
            index=idx.key,
        )
        field = sy.icd.DelimitedTextField(name="setpoint", position=1)
        message = sy.icd.Message(
            name="setpoint",
            payload=sy.icd.TextPayload(prefix="SET", fields=[field]),
        )
        ic = self.client.icds.create(
            sy.icd.ICD(
                name=SerialSim.device_name,
                messages=[message],
                framing=sy.icd.DelimitedFraming(),
            )
        )
        self.track_test_icds([ic])
        return serial.WriteTask(
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

    def _channel_keys(self, task: serial.WriteTask) -> list[int]:
        return [
            b.channel
            for t in task.config.targets
            for b in t.message.bindings
            if isinstance(b, sy.icd.ChannelWriteBinding)
        ]

    def assert_written(self) -> None:
        """Check that the lines since the last check end with LINES. A retried round
        sends its lines again before them."""

        def written() -> list[str] | None:
            lines = SerialSim.received()[self._checked :]
            return lines if lines[-len(self.LINES) :] == self.LINES else None

        lines = sy.poll(written, timeout=self.COMMAND_TIMEOUT, interval=0.1)
        if lines is None:
            new = SerialSim.received()[self._checked :]
            self.fail(f"the simulator got {new}, not {self.LINES} at the end")
        self._checked += len(lines)
