#  Copyright 2026 Synnax Labs, Inc.
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
1. The Core refuses to delete a device that a task uses
2. Task operation after the refused delete
3. Simulator crash and restart
4. Task operation after simulator restart
"""

from abc import ABC

import synnax as sy
from tests.driver.modbus_read import ModbusReadMixed
from tests.driver.opcua_read import OPCUAReadMixed
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, ReadTaskT


class DisconnectTask(SimulatorCase, ReadTaskCase[ReadTaskT], ABC):
    """
    Abstract base class providing disconnect/reconnect test behavior.

    Inherits from SimulatorCase and TaskCase to access simulator management and
    common test utilities. Overrides the run() method to execute a
    disconnect/reconnect test sequence.

    Usage:
        class DisconnectModbus(DisconnectTask, ModbusRead):
            pass

    The class uses these methods from TaskCase:
    - self.assert_sample_count(): Verify task operation
    - self.assert_device_exists(): Verify device existence
    - self.fail(): Fail the test with a message
    - self.log(): Log test progress
    - self.cleanup_simulator(): Kill simulator process
    - self.start_simulator(): Start simulator process

    The class expects these attributes from the task-specific class (e.g., ModbusRead):
    - self.tsk: Configured task instance
    - self.client: Synnax client
    - self.simulator_process: Running simulator process
    """

    def run(self) -> None:
        """Execute the disconnect/reconnect test sequence."""
        if self.tsk is None:
            self.fail("Task not configured. Check setup() in base class.")

        client = self.client
        tsk = self.tsk
        device = client.devices.retrieve(name=self.device_name)

        self.log("Test 1 - Delete Device Used by Task")
        try:
            client.devices.delete([device.key])
        except sy.ValidationError as e:
            assert "because task" in str(e), f"Refusal should name the task, got {e}"
            self.log(f"  Correctly refused: {e}")
        else:
            self.fail("Core deleted a device that a task uses")
        self.assert_device_exists(device_key=device.key)

        self.log("Test 2 - Run Task After Refused Delete")
        client.tasks.configure(tsk)
        self.assert_sample_count(task=tsk, strict=False)

        self.log("Test 3 - Kill Simulator")
        if self.simulator_process is None:
            self.fail("Simulator process not found")
        self.cleanup_simulator()

        self.log("Test 4 - Restart Simulator")
        self.start_simulator()

        self.log("Test 5 - Run Task")
        client.tasks.configure(tsk)
        self.assert_sample_count(task=tsk, strict=False)


class DisconnectOpcua(DisconnectTask[sy.opcua.ReadTask], OPCUAReadMixed):
    """
    OPC UA disconnect/reconnect test.
    """


class DisconnectModbus(DisconnectTask[sy.modbus.ReadTask], ModbusReadMixed):
    """
    Modbus TCP disconnect/reconnect test.
    """
