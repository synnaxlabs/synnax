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
import time

client = sy.Synnax()

dev = client.hardware.devices.retrieve(model="USB-6289")

volts_idx = client.channels.create(
    name="USB-6289 Time",
    is_index=True,
    retrieve_if_name_exists=True,
    data_type=sy.DataType.TIMESTAMP,
)
volts_1 = client.channels.create(
    name="USB-6289 Voltage 1",
    index=volts_idx.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

tsk = ni.AnalogReadTask(
    name="USB-6289 Analog Read",
    device=dev.key,
    sample_rate=sy.Rate.HZ * 100,
    stream_rate=sy.Rate.HZ * 25,
    data_saving=True,
    channels=[ni.AIVoltageChan(channel=volts_1.key, port=0)],
)

client.hardware.tasks.configure(tsk)
tsk.start()
time.sleep(5)
tsk.stop()
