#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Grand Finale: concurrent multi-protocol driver test.

Launches both OPC UA and Modbus simulators, creates read and write tasks across
both protocols, runs them all concurrently, and verifies data acquisition and
command delivery.

Channel creation is delegated to the existing concrete test case classes
(e.g. ModbusReadInputRegister.create_channels) to avoid duplication.
"""

from contextlib import ExitStack

import synnax as sy
from examples.modbus import ModbusSim
from examples.opcua import OPCUASim
from synnax import modbus, opcua

from tests.driver.modbus_read import ModbusReadCoil, ModbusReadInputRegister
from tests.driver.modbus_write import ModbusWriteCoil
from tests.driver.opcua_read import OPCUAReadMixed
from tests.driver.opcua_write import OPCUAWriteFloat
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import assert_sample_counts_in_range, assert_streamed_values


class GrandFinale(SimulatorCase):
    """Concurrent multi-protocol read and write task test."""

    sim_classes = [OPCUASim, ModbusSim]
    SAMPLE_RATE = 50 * sy.Rate.HZ
    STREAM_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 2 * sy.TimeSpan.SECOND

    def setup(self) -> None:
        self.read_tasks: list[sy.Task] = []
        self.write_tasks: list[sy.Task] = []
        super().setup()
        self._create_tasks()

    @property
    def all_tasks(self) -> list[sy.Task]:
        return self.read_tasks + self.write_tasks

    def _create_tasks(self) -> None:
        modbus_device = self.client.devices.retrieve(name=ModbusSim.device_name)
        opcua_device = self.client.devices.retrieve(name=OPCUASim.device_name)

        self.read_tasks = [
            modbus.ReadTask(
                name="GF Modbus Input Register",
                device=modbus_device.key,
                sample_rate=self.SAMPLE_RATE,
                stream_rate=self.STREAM_RATE,
                data_saving=True,
                channels=ModbusReadInputRegister.create_channels(self.client),
            ),
            modbus.ReadTask(
                name="GF Modbus Coil",
                device=modbus_device.key,
                sample_rate=self.SAMPLE_RATE,
                stream_rate=self.STREAM_RATE,
                data_saving=True,
                channels=ModbusReadCoil.create_channels(self.client),
            ),
            opcua.ReadTask(
                name="GF OPC UA Mixed",
                device=opcua_device.key,
                sample_rate=self.SAMPLE_RATE,
                stream_rate=self.STREAM_RATE,
                data_saving=True,
                array_mode=False,
                channels=OPCUAReadMixed.create_channels(self.client),
            ),
        ]

        self.write_tasks = [
            modbus.WriteTask(
                name="GF Modbus Write",
                device=modbus_device.key,
                channels=ModbusWriteCoil.create_channels(self.client),
            ),
            opcua.WriteTask(
                name="GF OPC UA Write",
                device=opcua_device.key,
                channels=OPCUAWriteFloat.create_channels(self.client),
            ),
        ]

        for task in self.all_tasks:
            self.client.tasks.configure(task)
            self.log(f"Configured task '{task.name}'")

    # ── Run ──────────────────────────────────────────────────────────

    def run(self) -> None:
        self.log(
            f"Grand Finale: {len(self.read_tasks)} read + "
            f"{len(self.write_tasks)} write = "
            f"{len(self.all_tasks)} concurrent tasks"
        )
        self._test_all_tasks_exist()
        with ExitStack() as stack:
            self._start_all_tasks(stack)
            self._test_read_data_flows()
            self._test_write_commands()
            self._test_read_sample_counts(stack)

    def _test_all_tasks_exist(self) -> None:
        self.log("Test 1 - Verify all tasks exist")
        for task in self.all_tasks:
            retrieved = self.client.tasks.retrieve(task.key)
            self.log(f"  {retrieved.name} (key={retrieved.key})")

    def _start_all_tasks(self, stack: ExitStack) -> None:
        self.log("Test 2 - Start all tasks concurrently")
        for task in self.all_tasks:
            stack.enter_context(task.run())
        self.log(f"  All {len(self.all_tasks)} tasks running")

    def _test_read_data_flows(self) -> None:
        self.log("Test 3 - Verify each read task is streaming data")
        for task in self.read_tasks:
            keys = self._task_channel_keys(task)
            with self.client.open_streamer(keys) as streamer:
                frame = streamer.read(timeout=5)
                if frame is None:
                    raise AssertionError(f"{task.name}: no data received within 5s")
            self.log(f"  {task.name}: streaming (OK)")

    def _test_write_commands(self) -> None:
        self.log("Test 4 - Send commands to write tasks")
        for task in self.write_tasks:
            cmd_keys = self._task_channel_keys(task)
            with self.client.open_streamer(cmd_keys) as streamer:
                with self.client.open_writer(
                    start=sy.TimeStamp.now(),
                    channels=cmd_keys,
                    name=f"{task.name}_gf_writer",
                ) as writer:
                    values = {k: float(42.0 + i) for i, k in enumerate(cmd_keys)}
                    writer.write(values)
                    assert_streamed_values(
                        self.client, streamer, values, task_name=task.name
                    )

                    values = {k: float(100.0 + i) for i, k in enumerate(cmd_keys)}
                    writer.write(values)
                    assert_streamed_values(
                        self.client, streamer, values, task_name=task.name
                    )
            self.log(f"  {task.name}: commands delivered (OK)")

    def _test_read_sample_counts(self, stack: ExitStack) -> None:
        self.log("Test 5 - Verify read sample counts")

        start_time = sy.TimeStamp.now()
        sy.sleep(self.TASK_DURATION.seconds * 1.25)
        stack.close()
        end_time = sy.TimeStamp.now()

        time_range = sy.TimeRange(start_time, end_time)
        for task in self.read_tasks:
            expected = int(task.config.sample_rate * self.TASK_DURATION.seconds)
            channel_keys = self._task_channel_keys(task)
            counts = assert_sample_counts_in_range(
                self.client,
                channel_keys=channel_keys,
                time_range=time_range,
                expected_samples=expected,
                task_name=task.name,
            )
            self.log(f"  {task.name}: {counts[0]} samples (OK)")

    # ── Helpers ──────────────────────────────────────────────────────

    @staticmethod
    def _task_channel_keys(task: sy.Task) -> list[int]:
        """Extract channel keys from a task, handling both read and write channels."""
        ch0 = task.config.channels[0]
        attr = "cmd_channel" if hasattr(ch0, "cmd_channel") else "channel"
        return [getattr(ch, attr) for ch in task.config.channels]

    # ── Teardown ─────────────────────────────────────────────────────

    def teardown(self) -> None:
        for task in self.all_tasks:
            try:
                self.client.tasks.delete(task.key)
            except sy.NotFoundError:
                self.log(f"Task '{task.name}' already deleted")
        super().teardown()
