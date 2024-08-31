#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from synnax.hardware import device
from synnax.hardware import rack
from synnax.hardware import task


class Client:
    devices: device.Client
    tasks: task.Client
    racks: rack.Client

    def __init__(
        self,
        devices: device.Client,
        racks: rack.Client,
        tasks: task.Client,
    ) -> None:
        self.tasks = tasks
        self.racks = racks
        self.devices = devices
