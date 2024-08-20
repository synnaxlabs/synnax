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

# dev = client.hardware.devices.retrieve(model=["USB-6000"])
# dev = [d for d in dev if d.model == "USB-6289"][0]
# print(dev)

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
volts_2 = client.channels.create(
    name="USB-6289 Voltage 2",
    index=volts_idx.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)
volts_3 = client.channels.create(
    name="USB-6289 Voltage 3",
    index=volts_idx.key,
    data_type=sy.DataType.FLOAT32,
    retrieve_if_name_exists=True,
)

tsk = ni.AnalogReadTask(
    name="USB-6289 Analog Read",
    device="123",
    sample_rate=sy.Rate.HZ * 200,
    stream_rate=sy.Rate.HZ * 25,
    data_saving=True,
    channels=[
        ni.AIVoltageChan(channel=volts_1.key, port=0),
        ni.AIVoltageChan(channel=volts_2.key, port=1),
        ni.AIVoltageChan(channel=volts_3.key, port=2),
    ]
)

# while True:
client.hardware.tasks.configure(tsk)
    # tsk.start()
    # time.sleep(5)
    # tsk.stop()
    # tsk.config.sample_rate *= 2
    # tsk.config.stream_rate *= 2
    # print(f"""
    # Sample Rate: {tsk.config.sample_rate}
    # Stream Rate: {tsk.config.stream_rate}
    # """)
    #
