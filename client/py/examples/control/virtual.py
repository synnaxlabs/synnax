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

ch = client.channels.create(
    name="signal",
    data_type="uint8",
    virtual=True,
    retrieve_if_name_exists=True
)

with client.control.acquire(
    name="Auto",
    read=["signal"],
    write=None
) as auto:
    auto.wait_until(lambda auto: auto["signal"] == 1)
    print("Signal received")
