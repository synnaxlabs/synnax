#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from datetime import datetime

import numpy as np

import synnax as sy

TEST_COUNT = 30

client = sy.Synnax()

test_time_channel_1 = client.channels.create(
    name="daq_4189_time",
    is_index=True,
    retrieve_if_name_exists=True,
)

test_time_channel_2 = client.channels.create(
    name="daq_9000_time",
    is_index=True,
    retrieve_if_name_exists=True,
)

test_channel_information = [
    {
        "name": "daq_4189_pressure",
        "count": 10,
        "data_type": sy.DataType.FLOAT32,
        "time_channel": test_time_channel_1.key,
    },
    {
        "name": "daq_4189_temperature",
        "count": 3,
        "data_type": sy.DataType.FLOAT32,
        "time_channel": test_time_channel_1.key,
    },
    {
        "name": "daq_9000_output",
        "count": 5,
        "data_type": sy.DataType.UINT8,
        "time_channel": test_time_channel_2.key,
    },
]

channels_to_create: list[sy.Channel] = []

for info in test_channel_information:
    for i in range(info["count"]):
        channels_to_create.append(
            sy.Channel(
                name=f"{info['name']}_{i}",
                data_type=info["data_type"],
                index=info["time_channel"],
            )
        )

client.channels.create(channels_to_create, retrieve_if_name_exists=True)

all_channels = [test_time_channel_1, test_time_channel_2] + channels_to_create

for i in range(1, TEST_COUNT + 1):
    test_tr = sy.TimeRange(
        start=datetime(2012, 1, i, 13, 0, 0),  # January i, 2012 1:00 PM
        end=datetime(2012, 1, i, 14, 45, 0),  # January i, 2012 3:00 PM
    )
    parent_tr = client.ranges.create(
        name=f"Test January {i}",
        time_range=test_tr,
        color="#E0E0E0",
        retrieve_if_name_exists=True,
    )
    child_tr_1 = parent_tr.create_sub_range(
        name="Pre-test",
        time_range=sy.TimeRange(
            start=datetime(2012, 1, i, 13, 0, 0),
            end=datetime(2012, 1, i, 13, 30, 0),
        ),
        color="#00FF00",
    )
    child_tr_2 = parent_tr.create_sub_range(
        name="Test Run",
        time_range=sy.TimeRange(
            start=datetime(2012, 1, i, 13, 30, 0),
            end=datetime(2012, 1, i, 14, 45, 0),
        ),
        color="#0000FF",
    )
    child_tr_3 = parent_tr.create_sub_range(
        name="Post-test",
        time_range=sy.TimeRange(
            start=datetime(2012, 1, i, 14, 45, 0),
            end=datetime(2012, 1, i, 15, 0, 0),
        ),
        color="#FF00FF",
    )

    # Generate data for each channel during the test run
    daq_4189_series: list[tuple[str, sy.Series]] = []
    daq_9000_series: list[tuple[str, sy.Series]] = []
    for channel in all_channels:
        if "daq_4189" in channel.name:
            if "time" in channel.name:
                # Generate timestamps at 100 Hz for the test run
                samples = int(sy.TimeSpan(test_tr.end - test_tr.start).seconds * 100)
                timestamps = np.linspace(
                    test_tr.start, test_tr.end, samples, dtype=np.int64
                )
                daq_4189_series.append(
                    (
                        channel.name,
                        sy.Series(
                            data=timestamps,
                            time_range=test_tr,
                            data_type=sy.DataType.TIMESTAMP,
                        ),
                    )
                )
                continue

            # Generate 100 Hz sinusoidal data
            samples = int(sy.TimeSpan(test_tr.end - test_tr.start).seconds * 100)
            t = np.linspace(
                0, sy.TimeSpan(test_tr.end - test_tr.start).seconds, samples
            )
            freq = np.random.uniform(0.1, 2.0)  # Random frequency between 0.1 and 2 Hz
            amp = np.random.uniform(0.5, 2.0)  # Random amplitude between 0.5 and 2
            offset = i + np.random.uniform(
                -0.5, 0.5
            )  # Center around i with some variation
            data = amp * np.sin(2 * np.pi * freq * t) + offset
            daq_4189_series.append(
                (
                    channel.name,
                    sy.Series(
                        data=data, time_range=test_tr, data_type=channel.data_type
                    ),
                )
            )

        elif "daq_9000" in channel.name:
            if "time" in channel.name:
                # Generate timestamps at 1 Hz for the test run
                samples = int(sy.TimeSpan(test_tr.end - test_tr.start).seconds)
                timestamps = np.linspace(
                    test_tr.start, test_tr.end, samples, dtype=np.int64
                )
                daq_9000_series.append(
                    (
                        channel.name,
                        sy.Series(
                            data=timestamps,
                            time_range=test_tr,
                            data_type=sy.DataType.TIMESTAMP,
                        ),
                    )
                )
                continue
            # Generate 1 Hz binary data
            samples = int(sy.TimeSpan(test_tr.end - test_tr.start).seconds)
            data = np.random.choice([0, 1], size=samples)

            daq_9000_series.append(
                (
                    channel.name,
                    sy.Series(
                        data=data, time_range=test_tr, data_type=channel.data_type
                    ),
                )
            )
    daq_4189_frame = sy.Frame({key: series for key, series in daq_4189_series})
    daq_9000_frame = sy.Frame({key: series for key, series in daq_9000_series})
    # Write the data to the channels
    client.write(test_tr.start, daq_4189_frame)
    client.write(test_tr.start, daq_9000_frame)
