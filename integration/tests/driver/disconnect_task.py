#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Task Disconnect/Reconnect Base Class

Provides generic disconnect/reconnect test behavior that can be mixed into
any task-specific test class via multiple inheritance.

This base class should be used with task-specific classes (ModbusRead, OpcuaRead, etc.)
to create disconnect tests like:

    class DisconnectModbus(DisconnectTask, ModbusRead):
        pass

The base class provides the run() method that tests:
1. Device deletion while task exists
2. Device reconnection
3. Task operation after reconnection
4. Simulator crash and restart
5. Task operation after simulator restart
"""

import synnax as sy

from tests.driver.task import Task


class DisconnectTask(Task):
    """
    Base class providing disconnect/reconnect test behavior.

    Inherits from Task to access common test utilities and infrastructure.
    Overrides the run() method to execute a disconnect/reconnect test sequence.

    Usage:
        class DisconnectModbus(DisconnectTask, ModbusRead):
            pass

    The class uses these methods from Task:
    - self.assert_sample_count(): Verify task operation
    - self.fail(): Fail the test with a message
    - self.log(): Log test progress
    - self._cleanup_simulator(): Kill simulator process
    - self._start_simulator(): Start simulator process

    The class expects these attributes from the task-specific class (e.g., ModbusRead):
    - self.tsk: Configured task instance
    - self.client: Synnax client
    - self.simulator_process: Running simulator process
    """

    def run(self) -> None:
        """Execute the disconnect/reconnect test sequence."""
        if self.tsk is None:
            self.fail("Task not configured. Check setup() in base class.")
            return

        client = self.client
        tsk = self.tsk
        device = client.hardware.devices.retrieve(key=tsk.config.device)

        self.log("Test 1 - Delete Device")
        try:
            client.hardware.devices.delete([device.key])
        except Exception as e:
            self.fail(f"Failed to delete device: {e}")
            return

        try:
            client.hardware.devices.retrieve(key=device.key)
            self.fail("Device still exists after deletion")
            return
        except sy.NotFoundError:
            pass  # Expected
        except Exception as e:
            self.fail(f"Unexpected error verifying device deletion: {e}")
            return

        self.log("Test 2 - Reconnect Device")
        try:
            reconnected_device = client.hardware.devices.create(device)
        except Exception as e:
            self.fail(f"Failed to recreate device: {e}")
            return

        try:
            retrieved_device = client.hardware.devices.retrieve(
                key=reconnected_device.key
            )
            if retrieved_device.name != device.name:
                self.fail(
                    f"Device name mismatch: {retrieved_device.name} != {device.name}"
                )
                return
        except Exception as e:
            self.fail(f"Failed to retrieve reconnected device: {e}")
            return

        self.log("Test 3 - Run Task After Device Reconnection")
        self.assert_sample_count(tsk, strict=False)


        self.log("Test 4 - Kill Simulator")
        if self.simulator_process is None:
            self.fail("Simulator process not found")
            return

        try:
            self._cleanup_simulator()
        except Exception as e:
            self.fail(f"Failed to kill simulator: {e}")
            return

        if self.simulator_process is not None:
            self.fail("Simulator process still running after cleanup")
            return

        self.log("Test 5 - Restart Simulator")
        try:
            self._start_simulator()
        except Exception as e:
            self.fail(f"Failed to restart simulator: {e}")
            return

        if self.simulator_process is None or self.simulator_process.poll() is not None:
            self.fail("Simulator process not running after restart")
            return

        self.log("Test 6 - Run Task")
        client.hardware.tasks.configure(tsk)
        self.assert_sample_count(tsk, strict=False)
