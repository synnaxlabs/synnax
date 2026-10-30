#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Modbus invalid configuration integration tests.

Each test saves a task with an invalid setting and verifies the driver rejects
it with an error status when the task is started.
"""

from examples.modbus import ModbusSim
from examples.modbus.server import STRING_START

import synnax as sy
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    DriverTask,
    assert_configure_rejected,
    assert_start_rejected,
    cleanup_task,
    create_channel,
    create_index,
    run_and_expect_rejection,
)


class ModbusInvalidConfig(SimulatorCase):
    """Verify the driver rejects invalid Modbus task configurations.

    Tests (run sequentially):
        1. Nonexistent device — device key that doesn't exist.
        2. Zero stream rate — heals to the schema default.
        3. Invalid rates — sample rate less than stream rate.
        4. Nonexistent channel key — Synnax channel that doesn't exist.
        5. Invalid address — register address the device doesn't serve.
        6. Duplicate channel — two tasks using the same channel.
        7. Wide string — a string longer than one request reads.
    """

    sim_classes = [ModbusSim]

    def setup(self) -> None:
        super().setup()
        self.device = self.client.devices.retrieve(name=self.device_name)

    def run(self) -> None:
        self.test_nonexistent_device()
        self.test_zero_stream_rate()
        self.test_invalid_rates()
        self.test_nonexistent_channel_key()
        self.test_invalid_address()
        self.test_duplicate_channel()
        self.test_wide_string()

    def test_nonexistent_device(self) -> None:
        """Configure a read task with a device key that doesn't exist."""
        self.log("Testing: Nonexistent device key")
        idx = create_index(self.client, "modbus_inv_dev_idx")
        task = sy.modbus.ReadTask(
            name="Modbus Invalid Device Test",
            device="nonexistent_device_key_12345",
            sample_rate=50 * sy.Rate.HZ,
            stream_rate=10 * sy.Rate.HZ,
            channels=[
                sy.modbus.InputRegisterReadChannel(
                    channel=create_channel(
                        self.client,
                        name="modbus_inv_dev_ch",
                        data_type=sy.DataType.UINT8,
                        index=idx.key,
                    ),
                    address=0,
                    data_type=sy.DataType.UINT8,
                ),
            ],
        )
        message = assert_configure_rejected(self.client, task, "nonexistent device")
        self.log(f"  Correctly rejected (nonexistent device): {message}")

    def test_zero_stream_rate(self) -> None:
        """Configure a read task with a stream rate of zero.

        A zero rate means unset, so the core heals it to the schema default
        instead of rejecting the task.
        """
        self.log("Testing: Zero stream rate heals to the schema default")
        idx = create_index(self.client, "modbus_inv_rate0_idx")
        task = sy.modbus.ReadTask(
            name="Modbus Zero Stream Rate Test",
            device=self.device.key,
            sample_rate=50 * sy.Rate.HZ,
            stream_rate=0,
            channels=[
                sy.modbus.InputRegisterReadChannel(
                    channel=create_channel(
                        self.client,
                        name="modbus_inv_rate0_ch",
                        data_type=sy.DataType.UINT8,
                        index=idx.key,
                    ),
                    address=0,
                    data_type=sy.DataType.UINT8,
                ),
            ],
        )
        self.client.tasks.configure(task)
        try:
            stored = self.client.tasks.retrieve(keys=[task.key])[0]
            rate = stored.config.get("stream_rate", stored.config.get("streamRate"))
            assert rate == 5, f"stream rate should heal to the default 5, got {rate}"
            self.log(f"  Healed to default stream rate: {rate}")
        finally:
            cleanup_task(self.client, task)

    def test_invalid_rates(self) -> None:
        """Start a read task with sample rate less than stream rate.

        The driver validates the rate relation when the task deploys.
        """
        self.log("Testing: Invalid rates (sample_rate < stream_rate)")
        idx = create_index(self.client, "modbus_inv_rate_idx")
        task = sy.modbus.ReadTask(
            name="Modbus Invalid Rate Test",
            device=self.device.key,
            sample_rate=10 * sy.Rate.HZ,
            stream_rate=100 * sy.Rate.HZ,
            channels=[
                sy.modbus.InputRegisterReadChannel(
                    channel=create_channel(
                        self.client,
                        name="modbus_inv_rate_ch",
                        data_type=sy.DataType.UINT8,
                        index=idx.key,
                    ),
                    address=0,
                    data_type=sy.DataType.UINT8,
                ),
            ],
        )
        self._assert_deploy_fails(task, "invalid rates")

    def test_nonexistent_channel_key(self) -> None:
        """Configure a read task with a Synnax channel key that doesn't exist."""
        self.log("Testing: Nonexistent Synnax channel key")
        task = sy.modbus.ReadTask(
            name="Modbus Invalid Channel Key Test",
            device=self.device.key,
            sample_rate=50 * sy.Rate.HZ,
            stream_rate=10 * sy.Rate.HZ,
            channels=[
                sy.modbus.InputRegisterReadChannel(
                    channel=999999999,
                    address=0,
                    data_type=sy.DataType.UINT8,
                ),
            ],
        )
        self._assert_deploy_fails(task, "nonexistent channel key")

    def test_invalid_address(self) -> None:
        """Start a read task with a register address the simulator doesn't serve."""
        self.log("Testing: Invalid register address (runtime)")
        idx = create_index(self.client, "modbus_inv_addr_idx")
        task = sy.modbus.ReadTask(
            name="Modbus Invalid Address Test",
            device=self.device.key,
            sample_rate=50 * sy.Rate.HZ,
            stream_rate=10 * sy.Rate.HZ,
            channels=[
                sy.modbus.InputRegisterReadChannel(
                    channel=create_channel(
                        self.client,
                        name="modbus_inv_addr_ch",
                        data_type=sy.DataType.UINT8,
                        index=idx.key,
                    ),
                    address=65000,
                    data_type=sy.DataType.UINT8,
                ),
            ],
        )
        message = assert_start_rejected(self.client, task, "invalid address")
        expected = (
            "Failed to read data: modbus_inv_addr_ch (input registers 65000): "
            "Illegal data address"
        )
        assert message == expected, f"Rejection should be {expected!r}, got {message!r}"

    def test_wide_string(self) -> None:
        """Start a read task with a string that needs more registers than the device
        reads in one request."""
        self.log("Testing: String wider than one request")
        rack = self.client.racks.retrieve(name=self.options.driver_rack)
        [narrow] = self.create_test_devices(
            [
                sy.modbus.Device(
                    host=ModbusSim.host,
                    port=ModbusSim.port,
                    name="Modbus Two Register Requests",
                    location=f"{ModbusSim.host}:{ModbusSim.port}",
                    rack=rack.key,
                    max_registers=2,
                )
            ]
        )
        idx = create_index(self.client, "modbus_inv_str_idx")
        task = sy.modbus.ReadTask(
            name="Modbus Wide String Test",
            device=narrow.key,
            sample_rate=50 * sy.Rate.HZ,
            stream_rate=10 * sy.Rate.HZ,
            channels=[
                sy.modbus.HoldingRegisterReadChannel(
                    channel=create_channel(
                        self.client,
                        name="modbus_inv_str_ch",
                        data_type=sy.DataType.STRING,
                        index=idx.key,
                    ),
                    address=STRING_START,
                    data_type=sy.DataType.STRING,
                    string_length=5,
                ),
            ],
        )
        message = assert_start_rejected(self.client, task, "wide string")
        expected = (
            "Failed to configure task: channels.0.string_length: needs 3 registers, "
            "but the device reads at most 2 per request"
        )
        assert message == expected, f"Rejection should be {expected!r}, got {message!r}"

    def test_duplicate_channel(self) -> None:
        """Configure and run two tasks that use the same channel."""
        self.log("Testing: Duplicate channel (two tasks on same channel)")
        idx = create_index(self.client, "modbus_dup_ch_idx")
        shared_ch_key = create_channel(
            self.client,
            name="modbus_dup_ch",
            data_type=sy.DataType.UINT8,
            index=idx.key,
        )

        def _make_task(name: str) -> sy.modbus.ReadTask:
            return sy.modbus.ReadTask(
                name=name,
                device=self.device.key,
                sample_rate=50 * sy.Rate.HZ,
                stream_rate=10 * sy.Rate.HZ,
                channels=[
                    sy.modbus.InputRegisterReadChannel(
                        channel=shared_ch_key,
                        address=0,
                        data_type=sy.DataType.UINT8,
                    ),
                ],
            )

        task_a = _make_task("Modbus Dup Channel Task A")
        task_b = _make_task("Modbus Dup Channel Task B")
        self.client.tasks.configure(task_a)
        self.log("  Task A configured")

        message: str | None = None
        try:
            with self.client.open_streamer([shared_ch_key]) as streamer, task_a.run():
                self.log("  Task A running")
                # A task takes the channel when its first sample arrives, not when it
                # starts. Wait for that, or Task B can take the channel first.
                if streamer.read(timeout=5) is None:
                    self.fail("Task A should write to the shared channel")
                self.log("  Task A writing")
                self.client.tasks.configure(task_b)
                self.log("  Task B configured (attempting run)")
                message = run_and_expect_rejection(self.client, task_b)
        finally:
            cleanup_task(self.client, task_a)
            cleanup_task(self.client, task_b)

        held = f"currently held by [{task_a.name}]"
        if message is None or held not in message:
            self.fail(f"Task B should fail as the channel is {held}, got {message!r}")
        self.log(f"  Correctly rejected on run: {message}")

    def _assert_deploy_fails(
        self, task: DriverTask[sy.task.StartConfig], label: str
    ) -> None:
        """Save a task and assert the driver rejects it on start."""
        message = assert_start_rejected(self.client, task, label)
        self.log(f"  Correctly rejected ({label}): {message}")
