#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
import sys
from pathlib import Path
from typing import Literal, Optional

import synnax as sy
from synnax.hardware import opcua

from framework.test_case import TestCase


class OPCUABasic(TestCase):
    """
    Test OPC UA Read Task integration.
    """

    server_process: Optional[subprocess.Popen[bytes]] = None

    def setup(self) -> None:
        """Launch OPC UA Server"""

        repo_root = Path(__file__).parent
        while repo_root.parent != repo_root:
            if (repo_root / ".git").exists():
                break
            repo_root = repo_root.parent
        server_script = repo_root / "driver" / "opc" / "dev" / "server_extended.py"

        if not server_script.exists():
            raise FileNotFoundError(f"OPC UA server script not found: {server_script}")

        # Launch the sim OPC server
        self.server_process = subprocess.Popen(
            [sys.executable, str(server_script)],
        )
        self.log(f"OPC UA server started with PID: {self.server_process.pid}")
        sy.sleep(1)
        if self.server_process.poll() is not None:
            raise RuntimeError(
                f"OPC UA server failed to start (exit code: {self.server_process.returncode})"
            )

    def run(self) -> None:
        """OPC UA Basic Test - Test all channel types (floats, arrays, bools)"""
        # Get or create device
        dev = self._get_device()
        SAMPLE_RATE = 100  # Hz
        TEST_DURATION = 3  # seconds

        # Create tasks
        f_task, f_chans = self._create_read_task(dev.key, "float", 8, SAMPLE_RATE)
        a_task, a_chans = self._create_read_task(dev.key, "array", 2, SAMPLE_RATE)
        b_task, b_chans = self._create_read_task(dev.key, "bool", 13, SAMPLE_RATE)
        all_tasks = [f_task, a_task, b_task]
        all_channels = f_chans + a_chans + b_chans

        self.log(f"=== Running all tasks for {TEST_DURATION} seconds ===")
        self._start_tasks(all_tasks)
        start = sy.TimeStamp.now()
        sy.sleep(TEST_DURATION)
        self._stop_tasks(all_tasks)
        end = sy.TimeStamp.now()
        self._delete_tasks(all_tasks)

        self._validate_data(all_channels, start, end, SAMPLE_RATE, TEST_DURATION)

    def _validate_data(
        self,
        channels: list[str],
        start: sy.TimeStamp,
        end: sy.TimeStamp,
        sample_rate: float,
        test_duration: int,
    ) -> None:
        """Validate that all channels have sufficient data.

        Args:
            channels: List of channel names to validate
            start: Start timestamp of data collection
            end: End timestamp of data collection
            sample_rate: Expected sample rate in Hz
            test_duration: Duration of data collection in seconds
        """
        self.log("=== Validating collected data ===")
        # Allow 80% tolerance for injected errors and latency
        expected_samples = sample_rate * test_duration * 0.8

        for channel_name in channels:
            tr = sy.TimeRange(start, end)
            ch = self.client.channels.retrieve(channel_name)
            data = ch.read(tr)
            sample_count = len(data)
            assert (
                sample_count >= expected_samples
            ), f"Channel {channel_name} has {sample_count} samples, expected at least {expected_samples}"

        self.log("Data validation passed for all channels")

    def _get_device(self) -> sy.Device:
        """Get the OPC UA test device."""
        client = self.client
        rack = client.hardware.racks.retrieve(name="Node 1 Embedded Driver")
        try:
            dev = client.hardware.devices.retrieve(key="opc-ua-test-server")
            self.log(f"Found existing device: {dev.name}")
        except:
            self.log("Creating new OPC UA device")
            dev = client.hardware.devices.create(
                sy.Device(
                    key="opc-ua-test-server",
                    rack=rack.key,
                    name="OPC UA Test Server",
                    make="OPC UA",
                    model="Test Server",
                    location="opc.tcp://localhost:4841",
                    properties='{"connection": {"endpoint": "opc.tcp://localhost:4841", "security_mode": "None", "security_policy": "None"}}',
                )
            )
            self.log(f"Created device: {dev.key}")
        return dev

    def _create_read_task(
        self,
        device_key: str,
        channel_type: Literal["float", "array", "bool"],
        base_node_id: int,
        sample_rate: float,
        count: int = 10,
    ) -> tuple[opcua.ReadTask, list[str]]:
        """Create an OPC UA Read Task with specified channel type.

        Args:
            device_key: Device key for the task
            channel_type: Type of channels ("float", "array", or "bool")
            base_node_id: Starting OPC UA NodeId number
            sample_rate: Sample rate in Hz
            count: Number of channels to create (default: 10)

        Returns:
            Tuple of (configured task, list of channel names for streaming)
        """
        # Map type-specific data types
        data_type = sy.DataType.UINT8 if channel_type == "bool" else sy.DataType.FLOAT32

        # Create index channel
        index_ch = self.client.channels.create(
            name=f"opcua_{channel_type}_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        # Create data channels
        channels = []
        channel_names = []
        for i in range(count):
            name = f"opcua_{channel_type}_{i}"
            ch = self.client.channels.create(
                name=name,
                data_type=data_type,
                index=index_ch.key,
                retrieve_if_name_exists=True,
            )
            channels.append(
                opcua.Channel(channel=ch.key, node_id=f"NS=2;I={base_node_id + i}")
            )
            channel_names.append(name)

        # Create and configure task
        array_mode = channel_type == "array"
        task = opcua.ReadTask(
            name=f"OPC UA Read - {channel_type}s",
            device=device_key,
            sample_rate=sy.Rate.HZ * sample_rate,
            stream_rate=sy.Rate.HZ * 10,
            data_saving=True,
            array_mode=array_mode,
            array_size=10 if array_mode else 1,
            channels=channels,
        )
        task.config.auto_start = True
        task = self.client.hardware.tasks.configure(task)
        self.log(f"Task configured: {task.name}")
        return task, channel_names

    def _start_tasks(self, tasks: list[opcua.ReadTask]) -> None:
        """Start a list of tasks."""
        self.log("Starting all tasks...")
        for task in tasks:
            task.start()

    def _stop_tasks(self, tasks: list[opcua.ReadTask]) -> None:
        """Stop a list of tasks."""
        self.log("Stopping all tasks...")
        for task in tasks:
            task.stop()

    def _delete_tasks(self, tasks: list[opcua.ReadTask]) -> None:
        """Delete a list of tasks."""
        self.log("Deleting all tasks...")
        for task in tasks:
            self.client.hardware.tasks.delete(task.key)

    def teardown(self) -> None:
        """Terminate OPC UA sim server."""
        if self.server_process is not None:
            self.log("Terminating OPC UA server...")
            self.server_process.terminate()
            try:
                self.server_process.wait(timeout=5)
                self.log("OPC UA server terminated successfully")
            except subprocess.TimeoutExpired:
                self.log("Server did not terminate gracefully, killing...")
                self.server_process.kill()
