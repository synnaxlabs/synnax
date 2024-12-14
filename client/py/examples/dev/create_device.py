#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

import synnax as sy

client = sy.Synnax()

rack = client.hardware.racks.create(name="Rack 1")

client.hardware.devices.create(
    [
        sy.Device(
            key="130227d9-02aa-47e4-b370-0d590add1bc1",
            rack=rack.key,
            name="PXI-6255",
            make="NI",
            model="PXI-6255",
            location="dev1",
            identifier="dev1",
        ),
        sy.Device(
            key="230227d9-02aa-47e4-b370-0d590add1bc1",
            rack=rack.key,
            name="LJM dtT4",
            make="LabJack",
            model="LJM_dtT4",
            location="dev2",
            identifier="dev2",
        ),
    ]
)
