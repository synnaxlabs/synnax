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
client = sy.Synnax(
    host="10.0.0.210",
    port=9090,
    username="synnax",
    password="seldon",
    secure=False
)
# Define the control channel names
NI_CHANNEL = "USB-6008_ai_0"
BOOL_CHANNEL = "test_bool"

bool_channel = client.channels.retrieve(BOOL_CHANNEL)

print("BOOL", bool_channel)

def print_out(auto):
    print(auto.state)
    return bool_channel.key in auto.state

# Open a control sequence under a context manager, so that the control is released when
# the block exits
with client.control.acquire(
    name="Press Sequence",
    write=[NI_CHANNEL, BOOL_CHANNEL],
    read =[NI_CHANNEL, BOOL_CHANNEL],
    write_authorities=[200],
) as ctrl:
    ctrl.set({
        BOOL_CHANNEL: True,
    })
    ctrl.wait_until(print_out)
    # ctrl.wait_until(
    #     lambda c: c[NI_CHANNEL] > 0.5,
    #     timeout=20 * sy.TimeSpan.SECOND,
    # )
    # print(ctrl[NI_CHANNEL])


# with client.control.acquire(
#     name="Press Sequence",
#     read =[ BOOL_CHANNEL],
#     write_authorities=[200],
# ) as ctrl:
#     ctrl.sleep(1)
#     ctrl.set({
#         BOOL_CHANNEL: True,
#     })
#     ctrl.wait_until_defined(BOOL_CHANNEL, timeout=20 * sy.TimeSpan.SECOND)
#     print("TEST_BOOL: ", ctrl[BOOL_CHANNEL])
  