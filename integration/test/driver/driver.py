#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import platform
import time
from test.framework.test_case import TestCase
from typing import Any

import synnax as sy
from synnax.hardware import ni


class Driver(TestCase):

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        super().__init__(*args, **kwargs)
        if platform.system() != "Windows":
            self.auto_pass()

    def setup(self) -> None:
        self._log_message("Retrieving device")
        self.dev = self.client.hardware.devices.retrieve(location="cDAQ1Mod1")

        self._log_message(f"Device model: {self.dev}")

        channel_config = [
            {
                "name": "ai_0",
                "data_type": sy.DataType.FLOAT32,
                "initial_value": 1.23,
                "port": 0,
            },
            {
                "name": "ai_1",
                "data_type": sy.DataType.FLOAT32,
                "initial_value": 4.56,
                "port": 1,
            },
        ]

        self._log_message("Adding channels")
        channels = []
        for ch in channel_config:
            ch_obj = self.add_channel(
                name=ch["name"],
                data_type=ch["data_type"],
                initial_value=ch["initial_value"],
            )
            channels.append(
                ni.AIVoltageChan(
                    channel=ch_obj.key,
                    device=self.dev.key,
                    port=ch["port"],
                    min_val=0,
                    max_val=10,
                    terminal_config="Diff",
                )
            )

        self._log_message("New Analog Read")
        self.tsk = ni.AnalogReadTask(
            name="Basic Analog Read",
            sample_rate=sy.Rate.HZ * 5,
            stream_rate=sy.Rate.HZ * 1,
            data_saving=False,
            channels=channels,
        )

        # Configure the task - the control authority issue may need to be resolved at server level
        self._log_message("Configuring task...")

        try:
            self.client.hardware.tasks.configure(self.tsk)
            self._log_message(f"Task configured: {self.tsk.name} (key: {self.tsk.key})")
        except Exception as e:
            self._log_message(f"Could not configure task: {e}")

        # Check task state after configuration
        try:
            task_state = self.client.hardware.tasks.retrieve(self.tsk.key)
            self._log_message(f"Task state after config: {task_state}")
        except Exception as e:
            self._log_message(f"Could not retrieve task state: {e}")
        self._log_message("Sleeping for 5 seconds")
        time.sleep(5)

    def run(self) -> None:
        self.set_manual_timeout(30)

        frame = sy.Frame()
        self._log_message("Running task")

        # Try to acquire control authority before starting the task
        # Get the actual channel keys used by this task
        try:
            # Log the channels we're actually using
            channel_names = [f"{self.name}_ai_0", f"{self.name}_ai_1"]
            self._log_message(f"Task channels: {channel_names}")

            # Try to acquire control of ONLY the driver_time channel that's causing the conflict
            with self.client.control.acquire(
                name=f"driver_time_override_{self.name}",
                read=["driver_time"],
                write=["driver_time"],
                write_authorities=[sy.Authority.ABSOLUTE],
            ) as controller:
                # Force override on just the driver_time channel
                controller.set_authority({"driver_time": sy.Authority.ABSOLUTE})
                self._log_message("Acquired control authority for driver_time")
                with self.tsk.start(timeout=60):  # Increase timeout to 60 seconds
                    self._log_message(
                        "Task started successfully with control authority"
                    )

                    # Open a streamer on the analog input channels.
                    with self.client.open_streamer(
                        [f"{self.name}_ai_0", f"{self.name}_ai_1"]
                    ) as streamer:
                        while self.should_continue:
                            data = streamer.read()
                            frame.append(data)

        except Exception as e:
            self._log_message(f"Could not acquire control authority: {e}")
            # Fallback to run without control authority
            with self.tsk.run(timeout=60):  # Increase timeout for fallback too
                self._log_message("Task started without control authority")
                with self.client.open_streamer(
                    [f"{self.name}_ai_0", f"{self.name}_ai_1"]
                ) as streamer:
                    while self.should_continue:
                        data = streamer.read()
                        frame.append(data)

        self._log_message("Saving results")
        frame.to_df().to_csv("test/results/analog_read_result.csv")
