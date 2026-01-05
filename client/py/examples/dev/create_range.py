#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import synnax as sy

# Lists for generating meaningful range names
experiments = ["Experiment", "Test", "Trial", "Analysis", "Measurement"]
subjects = ["Pressure", "Temperature", "Velocity", "Acceleration", "Force", "Voltage"]
conditions = ["High", "Low", "Medium", "Controlled", "Variable"]
locations = ["Lab1", "Chamber2", "Station3", "Unit4", "Bay5"]

client = sy.Synnax()
start_time = sy.TimeStamp.now()

# Create 100 ranges with random but relevant names
for i in range(100):
    # Generate a random name by combining words from our lists
    name = f"{random.choice(experiments)}_{random.choice(subjects)}_{random.choice(conditions)}_{random.choice(locations)}_{i+1}"

    # Create a time range that's sequential (each range starts after the previous one)
    range_start = start_time + (i * 10 * sy.TimeSpan.SECOND)
    range_end = range_start + 10 * sy.TimeSpan.SECOND

    # Generate a random color
    color = f"#{random.randint(0, 0xFFFFFF):06x}"

    client.ranges.create(
        name=name,
        time_range=sy.TimeRange(range_start, range_end),
        color=color,
    )
