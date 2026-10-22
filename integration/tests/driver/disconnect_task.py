#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Task Disconnect/Reconnect Base Classes

DisconnectTask adds disconnect/reconnect behavior to a read task case through multiple
inheritance:

    class DisconnectModbus(DisconnectTask, ModbusRead):
        pass

Its run() method tests:
1. Device deletion while task exists
2. Device reconnection
3. Task operation after reconnection
4. Simulator crash and restart
5. Task operation after simulator restart

BusDisconnect does the same for the CAN, serial, TCP, and UDP read cases, and checks
the warnings the task reports while its device is gone.
"""

from abc import ABC

import synnax as sy
from synnax import library
from tests.driver.bus_task import BusCase
from tests.driver.can_read import CANRead
from tests.driver.modbus_read import ModbusReadMixed
from tests.driver.opcua_read import OPCUAReadMixed
from tests.driver.serial_read import SerialRead
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, ReadTaskT
from tests.driver.tcp_read import TCPRead
from tests.driver.udp_read import UDPRead

RECONNECTING = "reconnecting"
MISSED = "missed 3 periods"
NOTHING_RECEIVED = "no frames received since start"
# Longer than the gap between two of any message the simulators send, so a message
# warns only once its simulator stops.
PERIOD = 200 * sy.TimeSpan.MILLISECOND
# A task warns that nothing arrived 10 s after it starts.
ABSENT_TIMEOUT = 20 * sy.TimeSpan.SECOND


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
    - self.assert_device_deleted(): Verify device deletion
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
            return

        client = self.client
        tsk = self.tsk
        device = client.devices.retrieve(name=self.device_name)

        self.log("Test 1 - Delete Device")
        client.devices.delete([device.key])
        self.assert_device_deleted(device_key=device.key)

        self.log("Test 2 - Reconnect Device")
        reconnected_device = client.devices.create(device)
        self.assert_device_exists(device_key=reconnected_device.key)
        sy.sleep(3)  # Give the driver time to reconnect

        self.log("Test 3 - Run Task After Device Reconnection")
        client.tasks.configure(tsk)
        self.assert_sample_count(task=tsk, strict=False)

        self.log("Test 4 - Kill Simulator")
        if self.simulator_process is None:
            self.fail("Simulator process not found")
            return
        self.cleanup_simulator()

        self.log("Test 5 - Restart Simulator")
        self.start_simulator()

        self.log("Test 6 - Run Task")
        client.tasks.configure(tsk)
        self.assert_sample_count(task=tsk, strict=False)

        client.devices.delete([reconnected_device.key])
        self.assert_device_deleted(device_key=reconnected_device.key)


class DisconnectOpcua(DisconnectTask[sy.opcua.ReadTask], OPCUAReadMixed):
    """
    OPC UA disconnect/reconnect test.
    """


class DisconnectModbus(DisconnectTask[sy.modbus.ReadTask], ModbusReadMixed):
    """
    Modbus TCP disconnect/reconnect test.
    """


def with_period(entries: list[library.Entry]) -> list[library.Entry]:
    """Returns entries with PERIOD on every message, so a task warns once a message
    stops arriving."""
    return [
        e.model_copy(update={"period": PERIOD})
        if isinstance(e, library.MessageEntry)
        else e
        for e in entries
    ]


class BusDisconnect(BusCase, ABC):
    """Takes away the device of a bus read task and checks that the task warns while
    the device is gone and reads again once it returns.

    The task must survive its device being deleted and created again. While the
    simulator is down, a running task warns with unplugged, and a task started then
    warns with absent.
    """

    unplugged: str
    absent: str

    def run(self) -> None:
        device = self.client.devices.retrieve(name=self.device_name)
        self.log("Delete and create the device")
        self.client.devices.delete([device.key])
        try:
            self.client.devices.retrieve(key=device.key)
        except sy.NotFoundError:
            pass
        else:
            raise AssertionError(f"{device.name} still exists after deletion")
        self.client.devices.create(device)
        self.client.tasks.configure(self.task)
        with self.statuses() as statuses, self.task.run():
            self.assert_resumes()
            self.assert_no_problems(statuses)
            self.log("Kill the simulator")
            self.cleanup_simulator()
            message = self.wait_for_status(
                statuses, "warning", containing=self.unplugged
            )
            self.log(f"Warning while unplugged: {message}")
        self.log("Start the task while the simulator is down")
        # A new streamer leaves out the warnings of the run before.
        with self.statuses() as statuses, self.task.run():
            message = self.wait_for_status(
                statuses, "warning", containing=self.absent, timeout=ABSENT_TIMEOUT
            )
            self.log(f"Warning while absent: {message}")
            self.log("Restart the simulator")
            self.start_simulator()
            self.assert_resumes()
            self.wait_for_status(statuses, "success")


class DisconnectTCP(BusDisconnect, TCPRead):
    """TCP disconnect/reconnect test."""

    unplugged = RECONNECTING
    absent = RECONNECTING


class DisconnectSerial(BusDisconnect, SerialRead):
    """Serial disconnect/reconnect test. The serial port disappears with its
    simulator."""

    unplugged = RECONNECTING
    absent = RECONNECTING


class DisconnectUDP(BusDisconnect, UDPRead):
    """UDP disconnect/reconnect test. The Driver's socket outlives the simulator, so
    the task warns about the messages that stop arriving."""

    unplugged = MISSED
    absent = NOTHING_RECEIVED

    def create_entries(self) -> list[library.Entry]:
        return with_period(super().create_entries())


class DisconnectCAN(BusDisconnect, CANRead):
    """CAN disconnect/reconnect test. The bus outlives the simulator, so the task warns
    about the messages that stop arriving."""

    unplugged = MISSED
    absent = NOTHING_RECEIVED

    def create_entries(self) -> list[library.Entry]:
        return with_period(super().create_entries())
