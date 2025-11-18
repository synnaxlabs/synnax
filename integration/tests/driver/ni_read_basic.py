#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import json
from synnax.hardware import ni

from framework.test_case import TestCase


class NIReadBasic(TestCase):
    """
    Test NI Analog Read Task integration.
    """

    def setup(self) -> None:
        """Setup for NI test (no external server needed)"""
        self.log("NI Analog Read Basic Test - Setup")

    def run(self) -> None:
        """NI Analog Read Basic Test - Test single voltage channel"""
        # Get or create device
        dev = self._get_device()
        SAMPLE_RATE = 100  # Hz
        TEST_DURATION = 10  # seconds

        # Create single analog read task with one voltage channel
        task, channel_names = self._create_read_task(dev.key, SAMPLE_RATE)

        self.log(f"=== Running task for {TEST_DURATION} seconds ===")
        self._start_task(task)
        start = sy.TimeStamp.now()
        sy.sleep(10)
        self._stop_task(task)
        end = sy.TimeStamp.now()
        self._delete_task(task)

        self._validate_data(channel_names, start, end, SAMPLE_RATE, TEST_DURATION)

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
        """Get the NI test device."""
        client = self.client
        rack = client.hardware.racks.retrieve(name="Node 1 Embedded Driver")
        try:
            dev = client.hardware.devices.retrieve(name="NI 9205")
            self.log(f"Found existing device: {dev.name}")
        except:
            self.log("Creating new NI device")
            dev = client.hardware.devices.create(
                sy.Device(
                    key="ni-test-device",
                    rack=rack.key,
                    name="NI 9205",
                    make="NI",
                    model="9205",
                    location="SYMod2AI",
                    properties='{"identifier": "SYMod2AI"}',
                )
            )
            self.log(f"Created device: {dev.key}")
        return dev

    def _create_read_task(
        self,
        device_key: str,
        sample_rate: float,
    ) -> tuple[sy.Task, list[str]]:
        """Create an NI Analog Read Task with a single voltage channel.

        Args:
            device_key: Device key for the task
            sample_rate: Sample rate in Hz

        Returns:
            Tuple of (configured task, list of channel names for streaming)
        """
        # Create index channel
        index_ch = self.client.channels.create(
            name="ni_analog_time",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )

        # Create single voltage channel
        channel_name = "ni_analog_voltage_0"
        ch = self.client.channels.create(
            name=channel_name,
            data_type=sy.DataType.FLOAT32,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

        # Create analog read task
        task = ni.AnalogReadTask(
            name="NI Analog Read - Voltage",
            data_saving=True,
            sample_rate=sy.Rate.HZ * sample_rate,
            stream_rate=sy.Rate.HZ * 25,
            channels=[
                ni.AIVoltageChan(
                    channel=ch.key,
                    device=device_key,
                    port=0,
                ),
            ],
        )
        self.client.hardware.tasks.configure(task)
        
        self.log("Running Task for 10s")
        task.start()
        sy.sleep(10)
        task.stop()
        
        tsk = ni.AnalogReadTask(self.client.hardware.tasks.retrieve(name="NI Analog Read - Voltage"))
        tsk.config.data_saving = False
        tsk.config.sample_rate = sy.Rate.HZ * 200
        self.client.hardware.tasks.configure(tsk)
        
        self.log("Running reconfifured Task")
        tsk.start()


        #self.log(f"Task configured: {task.name}")
        return updated_task, [channel_name]

    def _start_task(self, task: sy.Task) -> None:
        """Start the task."""
        self.log("Starting task...")
        task.execute_command_sync("start")

    def _stop_task(self, task: sy.Task) -> None:
        """Stop the task."""
        self.log("Stopping task...")
        task.execute_command_sync("stop")

    def _delete_task(self, task: sy.Task) -> None:
        """Delete the task."""
        self.log("Deleting task...")
        self.client.hardware.tasks.delete(task.key)

    def teardown(self) -> None:
        """Cleanup (NI doesn't need external server termination)"""
        self.log("NI Analog Read Basic Test - Teardown complete")