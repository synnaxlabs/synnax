#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from uuid import uuid4

client = sy.Synnax()

rack_key = 65537

client.devices.create(
    [
        sy.Device(
            key="pxi-6255",
            rack=65537,
            name="PXI-6255",
            make="NI",
            model="PXI-6255",
            location="dev1",
        ),
        sy.Device(
            key="labjack-t4",
            rack=65537,
            name="LabJack T4",
            make="LabJack",
            model="LJM_dtT4",
            location="dev2",
        ),
        sy.Device(
            key="ni-usb",
            rack=65537,
            name="USB-6000",
            make="NI",
            model="USB-6000",
            location="dev3",
        ),
        sy.Device(
            key="ethercat",
            rack=65537,
            make="EtherCAT",
            model="Beckhoff 1005",
            name="Beckhoff 1005",
            location="dev4"
        )
    ]
)
