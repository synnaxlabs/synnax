#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example simulates a basic control sequence where a tank with pressure represented
by press_pt is pressurized in 20 psi increments. The tank is then vented after holding
down to 5 psi after a period of time.

This script requires the `simulated_daq.py` script to be running in order to simulate
the data acquisition system (DAQ).

This script can be visualized in the Synnax Console by connecting the press_pt channel
to a line plot and making a schematic that shows valves representing press_vlv_cmd and
vent_vlv_cmd and a tank with a value representing press_pt.
"""

import synnax as sy

# We've logged in via the command-line interface, so there's no need to provide
# credentials here. See https://docs.synnaxlabs.com/reference/client/quick-start.
client = sy.Synnax()

# Define the control channel names
PRESS_VALVE = "press_vlv_cmd"
VENT_VALVE = "vent_vlv_cmd"
PRESSURE = "press_pt"

# Open a control sequence under a context manager, so that the control is released when
# the block exits
with client.control.acquire(
    # A useful name that identifies the sequence to the rest of the system. We highly
    # recommend keeping these names unique across your sequences.
    name="Pressurization Sequence",
    # Defines the authorities at which the sequence controls the valve channels. This is
    # a number from 0 to 255. A writer with a higher control authority can override a
    # writer with a lower control control authority.
    write_authorities=[200],
    # We need to set the channels we'll be writing to and reading from.
    write=[PRESS_VALVE, VENT_VALVE],
    read=[PRESSURE],
) as ctrl:
    # Mark the start of the sequence
    start = sy.TimeStamp.now()

    # Set the initial target pressure
    target_pressure = 20

    # Close the vent valve
    ctrl[VENT_VALVE] = False

    # Pressurize the system five times in 20 psi increments
    for i in range(5):
        # Open the pressurization valve
        ctrl[PRESS_VALVE] = True
        if ctrl.wait_until(
            # Wait until the pressure is greater than the current target
            lambda c: c[PRESSURE] > target_pressure,
            # If the pressure doesn't reach the target in 20 seconds, break the loop and
            # vent the system
            timeout=20 * sy.TimeSpan.SECOND,
        ):
            # Close the pressurization valve
            ctrl[PRESS_VALVE] = False
            # Wait for 2 seconds
            ctrl.sleep(2)
            # Increment the target
            target_pressure += 20
        else:
            break

    # Vent the system
    ctrl[VENT_VALVE] = True

    # Wait until the pressure is less than 5 psi
    ctrl.wait_until(lambda c: c[PRESSURE] < 5)

    # Close the vent valve
    ctrl[VENT_VALVE] = False

    # Mark the end of the sequence
    end = sy.TimeStamp.now()

    # Label the sequence with the end time
    client.ranges.create(
        name=f"Auto Pressurization Sequence {end}",
        time_range=sy.TimeRange(start=start, end=end),
    )
