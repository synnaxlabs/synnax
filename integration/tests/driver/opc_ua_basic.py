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
from typing import Optional
import synnax as sy
from synnax.hardware import opcua


from framework.test_case import TestCase


class OPCUABasic(TestCase):
    """
    Test OPC UA Read Task integration.
    """
    server_process: Optional[subprocess.Popen] = None

    def setup(self) -> None:
        """Launch OPC UA Server"""

        # Find the server script path relative to the integration tests directory
        # integration/tests/driver -> ../../driver/opc/dev/server_extended.py
        current_dir = Path(__file__).parent
        server_script = (
            current_dir / ".." / ".." / ".." / "driver" / "opc" / "dev" / "server_extended.py"
        ).resolve()

        if not server_script.exists():
            self.log(f"Server script not found at: {server_script}")
            raise FileNotFoundError(f"OPC UA server script not found: {server_script}")

        # Launch the sim OPC server
        self.server_process = subprocess.Popen(
            [sys.executable, str(server_script)],
        )
        self.log(f"OPC UA server started with PID: {self.server_process.pid}")
        sy.sleep(1)
        if self.server_process.poll() is not None:
            raise RuntimeError(f"OPC UA server failed to start (exit code: {self.server_process.returncode})")

    def run(self) -> None:
        """OPC UA Basic Test"""
        client = self.client
        rack = client.hardware.racks.retrieve(name="Node 1 Embedded Driver")

        try:
            dev = client.hardware.devices.retrieve(key="opc-ua-test-server")
            self.log(f"Found existing OPC UA device: {dev.name} (key={dev.key})")
        except Exception as e:
            self.log(f"Creating new OPC UA device")
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
            self.log(f"Created new OPC UA device: {dev.key}")

        # Create index channel for timestamps
        self.log("Creating channels")
        opcua_time = client.channels.create(
            name="opcua_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        # Create data channels for floats from the server
        my_float_0 = client.channels.create(
            name="my_float_0",
            data_type=sy.DataType.FLOAT32,
            index=opcua_time.key,
            retrieve_if_name_exists=True,
        )

        my_float_1 = client.channels.create(
            name="my_float_1",
            data_type=sy.DataType.FLOAT32,
            index=opcua_time.key,
            retrieve_if_name_exists=True,
        )

        # Create OPC UA Read Task
        self.log("Creating OPC UA Read Task")
        tsk = opcua.ReadTask(
            name="Test OPC UA Read",
            device=dev.key,
            sample_rate=sy.Rate.HZ * 10,  # Sample at 10 Hz (slower for testing)
            stream_rate=sy.Rate.HZ * 2,  # Stream at 2 Hz (5 samples per frame)
            data_saving=True,
            channels=[
                opcua.Channel(
                    channel=my_float_0.key,
                    node_id="NS=2;I=8",  # Numeric node ID for my_float_0
                ),
                opcua.Channel(
                    channel=my_float_1.key,
                    node_id="NS=2;I=9",  # Numeric node ID for my_float_1
                ),
            ],
        )

        tsk = client.hardware.tasks.configure(tsk)
        self.task = tsk
        self.log(f"Task configured with key: {tsk.key}")
        
        with tsk.run(timeout=3):
            # "Refresh" the connection by starting and immediately stopping the task
            # This forces the driver to acquire a connection from the pool
            # If the connection is stale, the pool will detect it and create a fresh one
            self.log("Refreshing OPC UA connection...")

        total_reads = 5
        frames_read = 0

        # Start task with fresh connection
        with tsk.run(timeout=30):
            self.log("Starting OPC Read task")
            with client.open_streamer(["my_float_0", "my_float_1"]) as streamer:
                for i in range(total_reads):
                    frame = streamer.read(timeout=2)
                    if frame is not None:
                        frames_read += 1
                        sample_count = len(frame["my_float_0"]) if "my_float_0" in frame else 0
                        print(frame)
                    else:
                        self.log(f"Frame {i + 1}/{total_reads}: No frame read")

        self.log("Stopping and Deleting task...")
        tsk.stop()
        client.hardware.tasks.delete(tsk.key)

        # Verify results
        assert (
            frames_read == total_reads
        ), f"Expected {total_reads} frames, got {frames_read}"

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
