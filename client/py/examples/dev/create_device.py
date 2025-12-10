#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

client = sy.Synnax()

rack = client.racks.create(name="NI / LabJack Test Rack")

client.devices.create(
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
            key="labjack-t4",
            rack=rack.key,
            name="LabJack T4",
            make="LabJack",
            model="LJM_dtT4",
            location="dev2",
            identifier="dev2",
        ),
        sy.Device(
            key="labjack-t7",
            rack=rack.key,
            name="LabJack T7",
            make="LabJack",
            model="LJM_dtT7",
            location="dev3",
            identifier="dev3",
        ),
        sy.Device(
            key="labjack-t8",
            rack=rack.key,
            name="LabJack T8",
            make="LabJack",
            model="LJM_dtT8",
            location="dev4",
            identifier="dev4",
        ),
        sy.Device(
            key="a0e37b26-5401-413e-8e65-c7ad9d9afd70",
            rack=rack.key,
            name="USB-6000",
            make="NI",
            model="USB-6000",
            location="dev3",
            identifier="dev3",
        ),
    ]
)
