#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from tests.driver.disconnect_task import DisconnectTask
from tests.driver.modbus_read import ModbusReadMixed


class DisconnectModbus(DisconnectTask, ModbusReadMixed):
    """
    Modbus TCP disconnect/reconnect test.

    Tests device and simulator disconnection/reconnection scenarios for
    Modbus TCP read tasks with mixed channel types. Inherits configuration
    from ModbusReadMixed and test behavior from DisconnectTask.

    The test sequence:
    1. Deletes the device while task exists
    2. Reconnects the device
    3. Verifies task operation after device reconnection
    4. Kills the simulator process
    5. Restarts the simulator process
    6. Verifies task operation after simulator restart
    """

    pass
