#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
import time

# We've logged in via the CLI, so there's no need to provide credentials here. See
# https://docs.synnaxlabs.com/reference/python-client/get-started for more information.
client = sy.Synnax()

# Define the control channel names
PRESS_VALVE = "valve_command_0"
VENT_VALVE = "valve_command_1"
PRESSURE = "sensor_0"

# Open a control sequence under a context manager, so that the control is released when
# the block exits
with client.control.acquire(
    name="Abort Sequence",
    write_authorities=[100],
    write=[PRESS_VALVE, VENT_VALVE],
    read=[PRESSURE],
) as controller:
    # Wait until we hit an abort condition.
    controller.wait_until(lambda c: c[PRESSURE] > 30)
    # Change the control authority to the highest level - 1, so the operator
    # can still take manual control from the schematic
    controller.set_authority({PRESS_VALVE: 254, VENT_VALVE: 254})
    # Vent the system
    controller.set({PRESS_VALVE: False, VENT_VALVE: True})
    # Hold control until the user presses Ctrl+C
    time.sleep(1e6)
