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
    host="nuc",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)

dev = client.hardware.devices.retrieve(model="USB-6289")

volts_idx = client.channels.create(
    name="6289_di_time",
    is_index=True,
    retrieve_if_name_exists=True,
    data_type=sy.DataType.TIMESTAMP,
)
di_0 = client.channels.create(
    name="6289_di_0",
    index=volts_idx.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)
di_1 = client.channels.create(
    name="6289_di_1",
    index=volts_idx.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

tsk = ni.DigitalReadTask(
    name="USB-6289 Digital Read",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 100,
    stream_rate=sy.Rate.HZ * 25,
    data_saving=True,
    channels=[
        ni.DIChan(channel=di_0.key, port=0, line=0),
        ni.DIChan(channel=di_1.key, port=0, line=1),
    ],
)

client.hardware.tasks.configure(tsk)
tsk.start()
total_reads = 100
frame = sy.Frame()
with client.open_streamer(["6289_di_0", "6289_di_1"]) as streamer:
    while total_reads > 0:
        frame.append(streamer.read())
        total_reads -= 1
frame.to_df().to_csv("digital_read_result.csv")
tsk.stop()
