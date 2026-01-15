#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload

from synnax.ontology import ID
from synnax.status import Status

DEVICE_ONTOLOGY_TYPE = ID(type="device")


def ontology_id(key: str) -> ID:
    """Returns the ontology ID for the Device entity."""
    return ID(type=DEVICE_ONTOLOGY_TYPE.type, key=key)


class DeviceStatusDetails(Payload):
    """Details about the status of a device."""

    rack: int = 0
    """The key of the rack the device is connected to."""
    device: str = ""
    """The key of the device."""


DeviceStatus = Status[DeviceStatusDetails]
"""The status of a device."""


class Device(Payload):
    key: str = ""
    location: str = ""
    rack: int = 0
    name: str = ""
    make: str = ""
    model: str = ""
    configured: bool = False
    properties: str = ""
    parent_device: str = ""
    status: DeviceStatus | None = None

    @property
    def ontology_id(self) -> ID:
        """Returns the ontology ID for this Device."""
        return ontology_id(self.key)
