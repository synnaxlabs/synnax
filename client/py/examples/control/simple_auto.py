#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

with client.control.acquire(
    name="Simple Automation",
    read=["pressure_1", "pressure_2", "temperature_1", "temperature_2"],
    write=["valve_1_cmd", "valve_2_cmd"],
) as auto:
    # Set initial valve states
    auto["valve_1_cmd"] = False

    # Wait until pressure_1 is less than 100 psi
    auto.wait_until(lambda auto: auto["pressure_1"] < 100)

    # Acquire absolute control on valve 2
    auto.authorize("valve_2_cmd", sy.Authority.ABSOLUTE)

    # Open valve 2
    auto["valve_2_cmd"] = True

    # Wait until pressure_2 is greater than 50 psi or temperature exceeds 100 degrees
    auto.wait_until(lambda auto: auto["pressure_2"] > 50 or auto["temperature_2"] > 100)

    # Close valve 2
    auto["valve_2_cmd"] = False

    # Wait until pressure_2 decreases below 50 psi
    auto.wait_until(lambda auto: auto["pressure_2"] < 50)

    # Close valve 1
    auto["valve_1_cmd"] = False
