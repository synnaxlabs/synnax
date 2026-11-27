#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys

from examples.serial_sim import SerialSim

import synnax as sy
from synnax import serial
from tests.driver.bus_task import BusCase
from tests.driver.task import ReadTaskCase, create_channel, create_index


class SerialRead(BusCase, ReadTaskCase[serial.ReadTask]):
    """Reads the temperature and pressure of each line that the serial simulator sends,
    stamped when the line arrives."""

    sim_classes = [SerialSim]
    task_name = "Serial Read"
    EVENT_DRIVEN = True
    # The rate at which the simulator sends lines.
    SAMPLE_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 3 * sy.TimeSpan.SECOND

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
    ) -> serial.ReadTask:
        idx = create_index(self.client, "serial_sim_time")
        channels = [
            create_channel(
                self.client,
                name=f"serial_sim_{name}",
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
                name=SerialSim.device_name,
                messages=[message],
                framing=sy.icd.DelimitedFraming(),
            )
        )
        self.track_test_icds([ic])
        return serial.ReadTask(
            name=task_name,
            device=device.key,
            entries=[
                sy.icd.ReadEntry(
                    icd=ic.key,
                    messages=[
                        sy.icd.ReadMessage(
                            message=message.key,
                            bindings=[
                                sy.icd.ReadBinding(field=f.key, channel=ch)
                                for f, ch in zip(fields, channels, strict=True)
                            ],
                        )
                    ],
                )
            ],
        )

    def _channel_keys(self, task: serial.ReadTask) -> list[int]:
        return [
            b.channel
            for e in task.config.entries
            for m in e.messages
            for b in m.bindings
        ]
