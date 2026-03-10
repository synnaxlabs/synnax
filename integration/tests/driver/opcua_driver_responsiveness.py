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

See: Desktop/Claude_Runbooks/OPCUA_Bug/opcua-connection-pool-bug.md
"""

import os
import signal
import subprocess
import threading

import synnax as sy

from tests.driver.opcua_task import OPCUAReadTaskCase
from tests.driver.task import create_channel, create_index

DEADLOCK_DURATION = 1 * sy.TimeSpan.SECOND
HEALTH_TIMEOUT = 5 * sy.TimeSpan.SECOND


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

    1. Start 4 tasks with a healthy sim (primes the connection pool).
    2. Stop all tasks (connections returned to pool as cached).
    3. SIGSTOP the sim (frozen process, TCP connections alive).
    4. Start all 4 tasks again — each calls pool.acquire().
    5. Health probe — task.stop() in a thread with wall-clock timeout.
       If the thread hangs, the driver is unresponsive.
    """

    task_name = "OPCUA Driver Responsiveness 0"
    NUM_TASKS = 4

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        return _create_read_channels(client, "concurrent_ss_0")

    def setup(self) -> None:
        super().setup()
        self.extra_tasks: list[sy.Task] = []
        self.frozen_pid: int | None = None
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

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Primary task not configured")
            return

        all_tasks = [self.tsk] + self.extra_tasks

        # Step 1-2: Prime the pool
        if not self._prime_pool(all_tasks):
            return

        # Step 3: Freeze the sim
        self._freeze_sim()

        # Step 4: Fire-and-forget start commands (don't wait for ack)
        self.log(f"Starting {len(all_tasks)} tasks with frozen sim")
        for task in all_tasks:
            task._internal.execute_command("start")

        sy.sleep(DEADLOCK_DURATION)

        # Step 5: Health probe — can the server still respond?
        self._check_server_responsive()

    def _prime_pool(self, tasks: list[sy.Task]) -> bool:
        self.log("Phase 1: Priming connection pool")
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

        self.log(f"Phase 1 complete: {len(tasks)} tasks primed")
        return True

    def _find_server_pid(self) -> int:
        """Find the actual process listening on the sim port via lsof."""
        assert self.sim is not None
        port = self.sim.port
        result = subprocess.run(
            ["lsof", "-ti", f":{port}"],
            capture_output=True,
            text=True,
        )
        pids = set(int(p) for p in result.stdout.strip().split("\n") if p)
        if not pids:
            raise RuntimeError(f"No process found listening on port {port}")
        # The LISTEN pid is the server; return it
        return pids.pop() if len(pids) == 1 else max(pids)

    def _freeze_sim(self) -> None:
        pid = self._find_server_pid()
        self.frozen_pid = pid
        self.log(f"Freezing simulator (SIGSTOP pid={pid})")
        os.kill(pid, signal.SIGSTOP)
        sy.sleep(0.5)

    def _check_server_responsive(self) -> None:
        self.log("Health probe: trying to stop a task (requires driver worker)")
        task = self.extra_tasks[0]
        result: list[bool] = [False]

        def try_stop() -> None:
            try:
                task.stop(timeout=HEALTH_TIMEOUT)
                result[0] = True
            except (TimeoutError, sy.UnexpectedError):
                result[0] = True

        t = threading.Thread(target=try_stop, daemon=True)
        t.start()
        t.join(timeout=HEALTH_TIMEOUT.seconds)

        if t.is_alive():
            self.fail(
                f"Driver unresponsive — task.stop() hung for "
                f"{HEALTH_TIMEOUT.seconds:.0f}s — deadlock confirmed"
            )
        else:
            self.log("Health probe OK — task responded")

    def teardown(self) -> None:
        # Resume the actual server process so parent teardown can terminate it
        if self.frozen_pid is not None:
            try:
                os.kill(self.frozen_pid, signal.SIGCONT)
            except OSError:
                pass
        for task in self.extra_tasks:
            try:
                self.client.tasks.delete(task.key)
            except sy.NotFoundError:
                pass
        super().teardown()
