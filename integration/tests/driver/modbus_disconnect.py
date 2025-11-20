#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Modbus Disconnect/Reconnect Task Test

This test validates that the Modbus read task can handle device
disconnection and reconnection scenarios gracefully.

Example JSON configuration:
{
  "case": "driver/modbus_disconnect"
}
"""

import synnax as sy

from tests.driver.modbus_read import ModbusRead


class ModbusDisconnect(ModbusRead):
    """
    Modbus TCP disconnect/reconnect test.

    This test validates that the Modbus read task can handle device
    disconnection and reconnection scenarios gracefully.
    """

    def run(self) -> None:
        """Execute the disconnect/reconnect test."""
        if self.tsk is None:
            self.fail("Task not configured")
            return

        client = self.client
        tsk = self.tsk
        device = client.hardware.devices.retrieve(key=tsk.config.device)
        
        try:
            self.log("Test 1 - Delete Device")
            client.hardware.devices.delete([device.key])
        except Exception as e:
            self.fail(f"Failed to delete device: {e}")
            return

        try:
            client.hardware.devices.retrieve(key=device.key)
            self.fail("Device still exists after deletion")
            return
        except sy.NotFoundError:
            pass
        except Exception as e:
            self.fail(f"Unexpected error retrieving device: {e}")

        self.log("Test 2 - Reconnect Device")
        try:
            reconnected_device = client.hardware.devices.create(device)
        except Exception as e:
            self.fail(f"Failed to recreate device: {e}")
            return

        try:
            retrieved_device = client.hardware.devices.retrieve(key=reconnected_device.key)
            if retrieved_device.name != device.name:
                self.fail(f"Device name mismatch: {retrieved_device.name} != {device.name}")
                return
        except Exception as e:
            self.fail(f"Failed to retrieve reconnected device: {e}")
            return

        self.log("Test 3 - Run Task")
        self.assert_sample_count(tsk, strict=False)

        self.log("Test 4 - Kill Simulator (While Task Running)")
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
    
        self.log("Test 5 - Restart Simulator Server")
        try:
            self._start_simulator()
        except Exception as e:
            self.fail(f"Failed to restart simulator: {e}")
            return

        # Verify simulator is running
        if self.simulator_process is None or self.simulator_process.poll() is not None:
            self.fail("Simulator process not running after restart")
            return


        self.log("Test 6 - Run Task")
        client.hardware.tasks.configure(tsk)
        self.assert_sample_count(tsk, strict=False)
