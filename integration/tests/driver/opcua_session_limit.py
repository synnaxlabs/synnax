#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
OPC UA session limit integration test.

Starts an OPC UA simulator with a restricted session limit, then exercises
the driver's connection pool under session pressure across three scenarios:

Test 1 (Exceed limit): With max_sessions=2, start 3 tasks. Two should
    succeed; the third should receive an error status because the server
    rejects the session.

Test 2 (Server restart): Kill the sim while tasks are running, restart it,
    then verify tasks recover and resume streaming — without exhausting
    sessions during reconnection.

Test 3 (Churn under pressure): With a tight session limit, repeatedly
    kill and restart the sim to trigger health-probe failures and
    connection churn. Verify all tasks eventually recover.
"""

from collections.abc import Generator
from contextlib import ExitStack

import synnax as sy
from examples.opcua import OPCUASim
from synnax.task.payload import Status

from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    assert_sample_counts_in_range,
    create_channel,
    create_index,
    send_and_verify_commands,
)


def _statuses_for_task(
    frame: sy.Frame, task_key: int
) -> Generator[Status, None, None]:
    """Yield all status updates for a specific task from a streamer frame."""
    if "sy_status_set" not in frame:
        return
    for i in range(len(frame["sy_status_set"])):
        status = Status.model_validate(frame["sy_status_set"][i])
        if status.details is not None and status.details.task == task_key:
            yield status


def _make_read_tasks(
    client: sy.Synnax,
    device_key: str,
    *,
    count: int,
    sample_rate: sy.Rate,
    stream_rate: sy.Rate,
) -> list[sy.Task]:
    """Create `count` independent OPC UA read tasks with unique channels."""
    tasks: list[sy.Task] = []
    for t in range(count):
        idx = create_index(client, f"sl_read_{t}_index")
        channels = [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"sl_read_{t}_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type="float32",
            )
            for i in range(2)
        ]
        tasks.append(
            sy.opcua.ReadTask(
                name=f"SL Read {t}",
                device=device_key,
                sample_rate=sample_rate,
                stream_rate=stream_rate,
                data_saving=True,
                array_mode=False,
                channels=channels,
            )
        )
    return tasks


def _make_write_task(
    client: sy.Synnax,
    device_key: str,
) -> sy.Task:
    """Create a single OPC UA write task with unique channels."""
    idx = create_index(client, "sl_write_cmd_time")
    channels = [
        sy.opcua.WriteChannel(
            cmd_channel=create_channel(
                client,
                name=f"sl_cmd_{i}",
                data_type=sy.DataType.FLOAT32,
                index=idx.key,
            ),
            node_id=f"NS=2;I={18 + i}",
        )
        for i in range(3)
    ]
    return sy.opcua.WriteTask(
        name="SL Write",
        device=device_key,
        channels=channels,
    )


def _task_channel_keys(task: sy.Task) -> list[int]:
    ch0 = task.config.channels[0]
    attr = "cmd_channel" if hasattr(ch0, "cmd_channel") else "channel"
    return [getattr(ch, attr) for ch in task.config.channels]


def _assert_streaming(client: sy.Synnax, task: sy.Task, timeout: int = 30) -> None:
    """Assert that a read task is producing data on its channels."""
    keys = _task_channel_keys(task)
    with client.open_streamer(keys) as streamer:
        frame = streamer.read(timeout=timeout)
        if frame is None:
            raise AssertionError(f"{task.name}: no data within {timeout}s")


# ── Test 1: Exceed session limit ─────────────────────────────────


class OPCUASessionLimitExceed(SimulatorCase):
    """Start more tasks than sessions available; verify the excess task
    receives an error status from the driver."""

    sim_classes = [OPCUASim]
    SAMPLE_RATE = 50 * sy.Rate.HZ
    STREAM_RATE = 10 * sy.Rate.HZ

    def setup(self) -> None:
        # 2 sessions, but we'll try to start 3 tasks.
        self.sims = {
            OPCUASim.device_name: OPCUASim(
                rate=self.SAMPLE_RATE,
                max_sessions=2,
            ),
        }
        super().setup()
        device = self.client.devices.retrieve(name=OPCUASim.device_name)
        self.tasks = _make_read_tasks(
            self.client,
            device.key,
            count=3,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=self.STREAM_RATE,
        )
        for task in self.tasks:
            self.client.tasks.configure(task)
            self.log(f"Configured '{task.name}'")

    def run(self) -> None:
        self.log("Test 1 - Exceed session limit (max_sessions=2, 3 tasks)")

        # Listen for status updates so we can detect the error.
        with self.client.open_streamer(["sy_status_set"]) as streamer:
            # Start all 3 tasks.
            stack = ExitStack()
            stack.__enter__()
            for task in self.tasks:
                stack.enter_context(task.run())

            # Collect status updates for a few seconds.
            error_tasks: set[int] = set()
            ok_tasks: set[int] = set()
            task_keys = {t.key for t in self.tasks}
            timer = sy.Timer()
            while timer.elapsed() < 10 * sy.TimeSpan.SECOND:
                frame = streamer.read(timeout=5)
                if frame is None:
                    continue
                for key in task_keys:
                    for status in _statuses_for_task(frame, key):
                        if status.variant == "error":
                            error_tasks.add(key)
                        elif (
                            status.variant == "success"
                            and status.details is not None
                            and status.details.running
                        ):
                            ok_tasks.add(key)
                # Once we've seen at least one error and the rest running,
                # we can stop early.
                if len(error_tasks) >= 1 and len(ok_tasks) >= 2:
                    break

            stack.close()

        if len(error_tasks) == 0:
            raise AssertionError(
                "Expected at least 1 task to fail with a session limit "
                "error, but all tasks started successfully. "
                f"ok={len(ok_tasks)}, errors={len(error_tasks)}"
            )

        self.log(
            f"  {len(ok_tasks)} tasks started OK, "
            f"{len(error_tasks)} rejected (expected)"
        )

    def teardown(self) -> None:
        for task in getattr(self, "tasks", []):
            try:
                self.client.tasks.delete(task.key)
            except sy.NotFoundError:
                pass
        super().teardown()


# ── Test 2: Server restart recovery ──────────────────────────────


class OPCUASessionLimitRestart(SimulatorCase):
    """With a tight session limit, kill the sim while tasks are running,
    restart it, and verify tasks recover and resume streaming."""

    sim_classes = [OPCUASim]
    SAMPLE_RATE = 50 * sy.Rate.HZ
    STREAM_RATE = 10 * sy.Rate.HZ

    def setup(self) -> None:
        # 4 sessions for 3 tasks — just enough room, no margin for churn.
        self.sims = {
            OPCUASim.device_name: OPCUASim(
                rate=self.SAMPLE_RATE,
                max_sessions=4,
            ),
        }
        super().setup()
        device = self.client.devices.retrieve(name=OPCUASim.device_name)
        self.read_tasks = _make_read_tasks(
            self.client,
            device.key,
            count=2,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=self.STREAM_RATE,
        )
        self.write_task = _make_write_task(self.client, device.key)
        self.all_tasks = self.read_tasks + [self.write_task]
        for task in self.all_tasks:
            self.client.tasks.configure(task)
            self.log(f"Configured '{task.name}'")

    def run(self) -> None:
        self.log("Test 2 - Server restart recovery (max_sessions=4)")

        with ExitStack() as stack:
            for task in self.all_tasks:
                stack.enter_context(task.run())

            # Verify initial streaming.
            for task in self.read_tasks:
                _assert_streaming(self.client, task)
            self.log("  All tasks streaming before restart")

            # Kill the simulator.
            self.log("  Killing simulator")
            self.cleanup_simulator()
            sy.sleep(3)

            # Restart with the same session limit.
            self.log("  Restarting simulator (max_sessions=4)")
            self.sims[OPCUASim.device_name] = OPCUASim(
                rate=self.SAMPLE_RATE,
                max_sessions=4,
            )
            self.sims[OPCUASim.device_name].start()
            self.sim = self.sims[OPCUASim.device_name]

            # Reconfigure tasks to trigger reconnection.
            for task in self.all_tasks:
                self.client.tasks.configure(task)

            # Give the driver time to reconnect.
            sy.sleep(3)

            # Verify tasks recovered.
            for task in self.read_tasks:
                _assert_streaming(self.client, task, timeout=30)
            self.log("  All tasks recovered after restart")

            # Verify write still works.
            send_and_verify_commands(
                self.client,
                cmd_keys=_task_channel_keys(self.write_task),
                writer_name="sl_restart_writer",
                task_name=self.write_task.name,
            )
            self.log("  Write task commands verified after restart")

    def teardown(self) -> None:
        for task in getattr(self, "all_tasks", []):
            try:
                self.client.tasks.delete(task.key)
            except sy.NotFoundError:
                pass
        super().teardown()


# ── Test 3: Churn under pressure ─────────────────────────────────


class OPCUASessionLimitChurn(SimulatorCase):
    """Repeatedly kill and restart the sim with a tight session limit to
    trigger connection churn from health-probe failures. Verify all tasks
    eventually recover after the final restart."""

    sim_classes = [OPCUASim]
    SAMPLE_RATE = 50 * sy.Rate.HZ
    STREAM_RATE = 10 * sy.Rate.HZ
    RESTART_CYCLES = 3

    def setup(self) -> None:
        self.sims = {
            OPCUASim.device_name: OPCUASim(
                rate=self.SAMPLE_RATE,
                max_sessions=4,
            ),
        }
        super().setup()
        device = self.client.devices.retrieve(name=OPCUASim.device_name)
        self.read_tasks = _make_read_tasks(
            self.client,
            device.key,
            count=3,
            sample_rate=self.SAMPLE_RATE,
            stream_rate=self.STREAM_RATE,
        )
        for task in self.read_tasks:
            self.client.tasks.configure(task)
            self.log(f"Configured '{task.name}'")

    def run(self) -> None:
        self.log(
            f"Test 3 - Churn under pressure "
            f"({self.RESTART_CYCLES} cycles, max_sessions=4)"
        )

        with ExitStack() as stack:
            for task in self.read_tasks:
                stack.enter_context(task.run())

            # Verify initial streaming.
            for task in self.read_tasks:
                _assert_streaming(self.client, task)
            self.log("  Initial streaming verified")

            # Repeatedly kill and restart to create churn.
            for cycle in range(self.RESTART_CYCLES):
                self.log(f"  Cycle {cycle + 1}/{self.RESTART_CYCLES}: kill")
                self.cleanup_simulator()
                # Let health probes fail and pool discard connections.
                sy.sleep(3)

                self.log(f"  Cycle {cycle + 1}/{self.RESTART_CYCLES}: restart")
                self.sims[OPCUASim.device_name] = OPCUASim(
                    rate=self.SAMPLE_RATE,
                    max_sessions=4,
                )
                self.sims[OPCUASim.device_name].start()
                self.sim = self.sims[OPCUASim.device_name]

                # Reconfigure to trigger reconnection.
                for task in self.read_tasks:
                    self.client.tasks.configure(task)
                sy.sleep(3)

            # After all cycles, verify every task recovers.
            self.log("  Verifying all tasks recovered after churn")
            for task in self.read_tasks:
                _assert_streaming(self.client, task, timeout=30)

            # Run for a sustained period and verify sample counts.
            start_time = sy.TimeStamp.now()
            duration = 2 * sy.TimeSpan.SECOND
            sy.sleep(duration.seconds * 1.25)

        end_time = sy.TimeStamp.now()
        time_range = sy.TimeRange(start_time, end_time)
        for task in self.read_tasks:
            expected = int(task.config.sample_rate * duration.seconds)
            counts = assert_sample_counts_in_range(
                self.client,
                channel_keys=_task_channel_keys(task),
                time_range=time_range,
                expected_samples=expected,
                task_name=task.name,
            )
            self.log(f"  {task.name}: {counts[0]} samples")

    def teardown(self) -> None:
        for task in getattr(self, "read_tasks", []):
            try:
                self.client.tasks.delete(task.key)
            except sy.NotFoundError:
                pass
        super().teardown()
