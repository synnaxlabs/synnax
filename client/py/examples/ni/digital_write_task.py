#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import time

import synnax as sy
from synnax.hardware import ni

client = sy.Synnax(
    host="nuc",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False,
)

dev = client.hardware.devices.retrieve(model="USB-6000")

do_1_cmd = client.channels.create(
    name="6289_do_1_cmd",
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
    virtual=True,
)

do_1_state_time = client.channels.create(
    name="6289_do_1_state_time",
    is_index=True,
    data_type=sy.DataType.TIMESTAMP,
    retrieve_if_name_exists=True,
)

do_1_state = client.channels.create(
    name="6289_do_1_state",
    index=do_1_state_time.key,
    data_type=sy.DataType.UINT8,
    retrieve_if_name_exists=True,
)

tsk = ni.DigitalWriteTask(
    name="USB-6289 Digital Write",
    device=dev.key,
    state_rate=sy.Rate.HZ * 100,
    data_saving=True,
    channels=[
        ni.DOChan(
            cmd_channel=do_1_cmd.key, state_channel=do_1_state.key, port=0, line=0
        ),
    ],
)

client.hardware.tasks.configure(tsk)

start = sy.TimeStamp.now()
with tsk.start():
    with client.control.acquire(
        name="Control Sequence",
        read=["6289_do_1_state"],
        write=["6289_do_1_cmd"],
        write_authorities=50,
    ) as auto:
        for i in range(5000):
            time.sleep(1)
            auto["6289_do_1_cmd"] = 1
            auto.wait_until(lambda c: c["6289_do_1_state"] == 1)
            time.sleep(1)
            auto["6289_do_1_cmd"] = 0
            auto.wait_until(lambda c: c["6289_do_1_state"] == 0)

client.hardware.tasks.delete(tsk.key)
print(sy.TimeStamp.since(start))
