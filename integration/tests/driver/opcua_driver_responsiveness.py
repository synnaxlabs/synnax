#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
OPC UA connection pool behavior with an unresponsive server.

Verifies that the driver remains responsive to non-OPC-UA tasks when an OPC UA
server becomes unresponsive. Uses a Modbus read task as the health probe —
if the driver can start, stream, and stop a Modbus task while OPC UA connections
are hung, the driver is responsive.
"""

import os
import signal
import sys
import threading

import psutil
import synnax as sy
from examples.modbus import ModbusSim
from synnax import modbus

from tests.driver.modbus_read import ModbusReadInputRegister
from tests.driver.opcua_task import OPCUAReadTaskCase
from tests.driver.task import create_channel, create_index

DEADLOCK_DURATION = 1 * sy.TimeSpan.SECOND
HEALTH_TIMEOUT = 30 * sy.TimeSpan.SECOND


def _create_read_channels(client: sy.Synnax, prefix: str) -> list[sy.opcua.ReadChannel]:
    idx = create_index(client, f"{prefix}_index")
    return [
        sy.opcua.ReadChannel(
            channel=create_channel(
                client,
                name=f"{prefix}_float_{i}",
                data_type=sy.DataType.FLOAT32,
                index=idx.key,
            ),
            node_id=f"NS=2;I={8 + i}",
            data_type="float32",
        )
        for i in range(2)
    ]


class OPCUADriverResponsiveness(OPCUAReadTaskCase):
    """Tests that the driver remains responsive when an OPC UA server
    becomes unresponsive with cached connections in the pool.

    The OPC UA connection pool bug can block all driver worker threads,
    making the entire driver unresponsive — not just OPC UA operations.

    1. Start both OPC UA and Modbus simulators.
    2. Start 4 OPC UA tasks with a healthy sim (primes the connection pool).
    3. Stop all OPC UA tasks (connections returned to pool as cached).
    4. SIGSTOP the OPC UA sim (frozen process, TCP connections alive).
    5. Start all 4 OPC UA tasks again — each calls pool.acquire().
    6. Health probe: start a Modbus read task, verify it streams data, stop it.
       If this works, the driver worker threads are NOT all blocked.
    """

    task_name = "OPCUA Driver Responsiveness 0"
    sim_classes = [OPCUAReadTaskCase.sim_classes[0], ModbusSim]
    NUM_TASKS = 4

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        return _create_read_channels(client, "concurrent_ss_0")

    def setup(self) -> None:
        super().setup()
        self.extra_tasks: list[sy.Task] = []
        self.frozen_pid: int | None = None
        self.modbus_task: sy.Task | None = None
        device = self.client.devices.retrieve(name=self.device_name)
        for i in range(1, self.NUM_TASKS):
            channels = _create_read_channels(self.client, f"concurrent_ss_{i}")
            task = sy.opcua.ReadTask(
                name=f"OPCUA Driver Responsiveness {i}",
                device=device.key,
                sample_rate=self.SAMPLE_RATE,
                stream_rate=self.STREAM_RATE,
                data_saving=True,
                channels=channels,
            )
            self.client.tasks.configure(task)
            self.extra_tasks.append(task)

        self._setup_modbus_probe()

    def _setup_modbus_probe(self) -> None:
        modbus_device = self.client.devices.retrieve(name=ModbusSim.device_name)
        channels = ModbusReadInputRegister.create_channels(self.client)
        self.modbus_task = modbus.ReadTask(
            name="Driver Responsiveness Modbus Probe",
            device=modbus_device.key,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=self.STREAM_RATE,
            data_saving=True,
            channels=channels,
        )
        self.client.tasks.configure(self.modbus_task)

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Primary task not configured")
            return

        all_tasks = [self.tsk] + self.extra_tasks

        # Step 1-2: Prime the pool
        if not self._prime_pool(all_tasks):
            return

        # Step 3: Freeze the OPC UA sim
        self._freeze_sim()

        # Step 4: Fire-and-forget start commands (don't wait for ack)
        self.log(f"Starting {len(all_tasks)} tasks with frozen sim")
        for task in all_tasks:
            task._internal.execute_command("start")

        sy.sleep(DEADLOCK_DURATION)

        # Step 5: Health probe — can the driver still run a Modbus task?
        self._check_driver_responsive()

    def _prime_pool(self, tasks: list[sy.Task]) -> bool:
        self.log("Priming task connection pool")
        for task in tasks:
            try:
                task.start(timeout=5)
                self.log(f"  Started {task.name}")
            except (TimeoutError, sy.UnexpectedError) as e:
                self.fail(f"Phase 1: failed to start {task.name}: {e}")
                return False

        for task in tasks:
            try:
                task.stop(timeout=5)
            except (TimeoutError, sy.UnexpectedError) as e:
                self.fail(f"Phase 1: failed to stop {task.name}: {e}")
                return False

        return True

    def _freeze_sim(self) -> None:
        assert self.sim is not None and self.sim.process is not None
        pid = self.sim.process.pid
        assert pid is not None
        self.frozen_pid = pid
        self.log(f"Freezing OPC UA simulator (pid={pid})")
        if sys.platform == "win32":
            psutil.Process(pid).suspend()
        else:
            os.kill(pid, signal.SIGSTOP)
        sy.sleep(0.5)

    def _check_driver_responsive(self) -> None:
        """Start a Modbus read task and verify it streams data.

        The entire probe runs in a daemon thread with a wall-clock timeout
        so the test never hangs if the driver is deadlocked.
        """
        modbus_task = self.modbus_task
        assert modbus_task is not None
        self.log("Health probe: starting Modbus read task")

        result: list[str] = []

        def probe() -> None:
            try:
                modbus_task.start(timeout=HEALTH_TIMEOUT.seconds)
                keys = [ch.channel for ch in modbus_task.config.channels]
                with self.client.open_streamer(keys) as streamer:
                    frame = streamer.read(timeout=HEALTH_TIMEOUT.seconds)
                    if frame is None:
                        result.append("Modbus task not streaming data")
                        return
                modbus_task.stop(timeout=HEALTH_TIMEOUT.seconds)
                result.append("ok")
            except Exception as e:
                result.append(str(e))

        t = threading.Thread(target=probe, daemon=True)
        t.start()
        t.join(timeout=HEALTH_TIMEOUT.seconds)

        if t.is_alive():
            self.fail(
                f"Driver unresponsive — health probe hung for "
                f"{HEALTH_TIMEOUT.seconds:.0f}s"
            )
        elif not result or result[0] != "ok":
            err = result[0] if result else "unknown"
            self.fail(f"Driver unresponsive — Modbus probe failed: {err}")
        else:
            self.log("Health probe OK — Modbus task started, streamed, stopped")

    def teardown(self) -> None:
        if self.frozen_pid is not None:
            try:
                if sys.platform == "win32":
                    psutil.Process(self.frozen_pid).resume()
                else:
                    os.kill(self.frozen_pid, signal.SIGCONT)
            except (OSError, psutil.NoSuchProcess, psutil.AccessDenied):
                pass
        if self.modbus_task is not None:
            try:
                self.client.tasks.delete(self.modbus_task.key)
            except sy.NotFoundError:
                pass
        for task in self.extra_tasks:
            try:
                self.client.tasks.delete(task.key)
            except sy.NotFoundError:
                pass
        super().teardown()
