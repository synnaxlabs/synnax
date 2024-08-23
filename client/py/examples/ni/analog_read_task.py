#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.hardware import ni

client = sy.Synnax(
    # host="nuc",
    # port=9090,
    # username="synnax",
    # password="seldon",
    # secure=False,
)

# dev = client.hardware.devices.retrieve(model="USB-6289")

volts_idx = client.channels.create(
    name="6289_ai_time",
    is_index=True,
    retrieve_if_name_exists=True,
    data_type=sy.DataType.TIMESTAMP,
)
ai_0 = client.channels.create(
    name="6289_ai_0",
    index=volts_idx.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
ai_1 = client.channels.create(
    name="6289_ai_1",
    index=volts_idx.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
ai_2 = client.channels.create(
    name="6289_ai_2",
    index=volts_idx.key,
    data_type=sy.DataType.FLOAT32,
)

tsk = ni.AnalogReadTask(
    name="USB-6289 Analog Read",
    device="",
    sample_rate=sy.Rate.HZ * 100,
    stream_rate=sy.Rate.HZ * 25,
    data_saving=True,
    channels=[
        ni.AIVoltageChan(
            channel=ai_0.key,
            port=0,
            custom_scale=ni.LinScale(
                slope=2e4,
                y_intercept=50,
                pre_scaled_units="Volts",
                scaled_units="Volts",
            )
        ),
        ni.AIVoltageChan(
            channel=ai_1.key,
            port=1,
            custom_scale=ni.MapScale(
                pre_scaled_min=0,
                pre_scaled_max=10,
                scaled_min=0,
                scaled_max=200,
                pre_scaled_units="Volts",
                scaled_units="Degrees",
            )
        ),
    ],
)

client.hardware.tasks.configure(tsk)
tsk.start()
total_reads = 100
frame = sy.Frame()
with client.open_streamer(["6289_ai_0", "6289_ai_1"]) as streamer:
    while total_reads > 0:
        frame.append(streamer.read())
        total_reads -= 1
frame.to_df().to_csv("analog_read_result.csv")
tsk.stop()
