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


class GrandFinale(SimulatorCase):
    """Concurrent multi-protocol read and write task test."""

    sim_classes = [OPCUASim, ModbusSim]
    SAMPLE_RATE = 50 * sy.Rate.HZ
    STREAM_RATE = 10 * sy.Rate.HZ
    TASK_DURATION = 2 * sy.TimeSpan.SECOND

    def setup(self) -> None:
        self.read_tasks: list[sy.Task] = []
        self.write_tasks: list[sy.Task] = []
        self._stack: ExitStack | None = None
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
        self.test_all_tasks_exist()
        self.test_start_all_tasks()
        self.test_read_data_flows()
        self.test_write_commands()
        self.test_read_sample_counts()

    def test_all_tasks_exist(self) -> None:
        self.log("Test 1 - Verify all tasks exist")
        for task in self.all_tasks:
            retrieved = self.client.tasks.retrieve(task.key)
            self.log(f"  {retrieved.name} (key={retrieved.key})")

    def test_start_all_tasks(self) -> None:
        self.log("Test 2 - Start all tasks concurrently")
        self._stack = ExitStack().__enter__()
        for task in self.all_tasks:
            self._stack.enter_context(task.run())
        self.log(f"  All {len(self.all_tasks)} tasks running")

    def test_read_data_flows(self) -> None:
        self.log("Test 3 - Verify each read task is streaming data")
        for task in self.read_tasks:
            keys = self._read_channel_keys(task)
            with self.client.open_streamer(keys) as streamer:
                frame = streamer.read(timeout=5)
                if frame is None:
                    raise AssertionError(f"{task.name}: no data received within 5s")
            self.log(f"  {task.name}: streaming (OK)")

    def test_write_commands(self) -> None:
        self.log("Test 4 - Send commands to write tasks")
        for task in self.write_tasks:
            cmd_keys = self._write_channel_keys(task)
            with self.client.open_streamer(cmd_keys) as streamer:
                with self.client.open_writer(
                    start=sy.TimeStamp.now(),
                    channels=cmd_keys,
                    name=f"{task.name}_gf_writer",
                ) as writer:
                    values = {k: float(42.0 + i) for i, k in enumerate(cmd_keys)}
                    writer.write(values)
                    self._assert_streamed_values(task.name, streamer, values)

                    values = {k: float(100.0 + i) for i, k in enumerate(cmd_keys)}
                    writer.write(values)
                    self._assert_streamed_values(task.name, streamer, values)
            self.log(f"  {task.name}: commands delivered (OK)")

    def test_read_sample_counts(self) -> None:
        self.log("Test 5 - Verify read sample counts")

        start_time = sy.TimeStamp.now()
        sy.sleep(self.TASK_DURATION.seconds * 1.25)
        if self._stack is not None:
            self._stack.__exit__(None, None, None)
        end_time = sy.TimeStamp.now()

        time_range = sy.TimeRange(start_time, end_time)
        for task in self.read_tasks:
            expected = int(task.config.sample_rate * self.TASK_DURATION.seconds)
            min_samples = int(expected * 0.60)
            max_samples = int(expected * 1.40)
            channel_keys = self._read_channel_keys(task)
            sample_counts = []

            for key in channel_keys:
                ch = self.client.channels.retrieve(key)
                n = len(ch.read(time_range))
                sample_counts.append(n)
                if n < min_samples or n > max_samples:
                    raise AssertionError(
                        f"{task.name}, channel '{ch.name}': "
                        f"{n} samples, expected {expected} ±40% "
                        f"({min_samples}-{max_samples})"
                    )

            if len(set(sample_counts)) > 1:
                raise AssertionError(
                    f"{task.name}: channels have different sample "
                    f"counts: {sample_counts}"
                )

            self.log(f"  {task.name}: {sample_counts[0]} samples (OK)")

    # ── Helpers ──────────────────────────────────────────────────────

    @staticmethod
    def _read_channel_keys(task: sy.Task) -> list[int]:
        return [ch.channel for ch in task.config.channels]

    @staticmethod
    def _write_channel_keys(task: sy.Task) -> list[int]:
        ch0 = task.config.channels[0]
        attr = "cmd_channel" if hasattr(ch0, "cmd_channel") else "channel"
        return [getattr(ch, attr) for ch in task.config.channels]

    def _assert_streamed_values(
        self,
        task_name: str,
        streamer: sy.Streamer,
        expected: dict[int, float],
        timeout: sy.TimeSpan = 5 * sy.TimeSpan.SECOND,
    ) -> None:
        received: dict[int, float] = {}
        timer = sy.Timer()
        while len(received) < len(expected):
            if timer.elapsed() > timeout:
                missing = set(expected.keys()) - set(received.keys())
                raise AssertionError(
                    f"{task_name}: timeout waiting for command "
                    f"values. Missing keys: {missing}"
                )
            frame = streamer.read(timeout=timeout)
            if frame is None:
                continue
            for key in expected:
                if key in frame and len(frame[key]) > 0:
                    received[key] = float(frame[key][-1])

        for key, exp_val in expected.items():
            if received[key] != exp_val:
                ch = self.client.channels.retrieve(key)
                raise AssertionError(
                    f"{task_name}, channel '{ch.name}': "
                    f"expected {exp_val}, got {received[key]}"
                )

    # ── Teardown ─────────────────────────────────────────────────────

    def teardown(self) -> None:
        if self._stack is not None:
            self._stack.__exit__(None, None, None)
            self._stack = None
        for task in self.all_tasks:
            try:
                self.client.tasks.delete(task.key)
            except sy.NotFoundError:
                pass
        super().teardown()
