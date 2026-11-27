#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import UnaryClient
from synnax.ni import device


class Client:
    """The clients of the NI resources in a Core."""

    devices: device.Client
    """Creates, retrieves, updates, and deletes NI devices."""

    def __init__(self, *, client: UnaryClient) -> None:
        self.devices = device.Client(client=client)
