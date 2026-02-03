#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
This example is a short example that demonstrates how to use a virtual channel to
communicate through a Synnax cluster. A channel named 'signal' is created, and when
another process (such as the Synnax Console) writes a value of 1 to this channel, the
script prints out 'Signal received'.
"""

import synnax as sy

client = sy.Synnax()

channel = client.channels.create(
    name="signal", data_type="uint8", virtual=True, retrieve_if_name_exists=True
)

with client.control.acquire(
    name="Signal Listener", read=["signal"], write=None
) as controller:
    controller.wait_until(lambda auto: auto["signal"] == 1)
    print("Signal received")
