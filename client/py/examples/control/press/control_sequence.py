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
    # A useful name that identifies the sequence to the rest of the system. We highly
    # recommend keeping these names unique across your sequences.
    name="Press Sequence",
    # Defines the authorities at which the sequence controls the valve channels. This is
    # a number from 0 to 255, where 0 is the lowest and 255 is the highest. It's up to
    # you to define what these mean in your system.
    write_authorities=[200],
    # We need to set the channels we'll be writing to and reading from.
    write=[PRESS_VALVE, VENT_VALVE],
    read=[PRESSURE],
) as controller:
    # Mark the start of the sequence
    start = sy.TimeStamp.now()

    # Close the vent valve
    controller[VENT_VALVE] = False

    # Set the initial target pressure
    curr_target = 20

    # Pressurize the system five times in 20 psi increments
    for i in range(5):
        # Open the pressurization valve
        controller[PRESS_VALVE] = True
        if controller.wait_until(
            # Wait until the pressure is greater than the current target
            lambda c: c[PRESSURE] > curr_target,
            # If the pressure doesn't reach the target in 20 seconds, break the loop and
            # vent the system
            timeout=20 * sy.TimeSpan.SECOND,
        ):
            # Close the pressurization valve
            controller[PRESS_VALVE] = False
            # Wait for 2 seconds
            time.sleep(2)
            # Increment the target
            curr_target += 20
        else:
            break

    # Mark the end of the sequence
    end = sy.TimeStamp.now()

    # Label the sequence with the end time
    client.ranges.create(
        name=f"Auto Pressurization Sequence {end}",
        time_range=sy.TimeRange(
            start=start,
            end=end,
        ),
    )
