#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from pydantic import BaseModel

from synnax import ontology, status

Key = str

ONTOLOGY_TYPE = ontology.ID(type="device")


def ontology_id(key: str) -> ontology.ID:
    """Returns the ontology ID for the Device entity."""
    return ontology.ID(type=ONTOLOGY_TYPE.type, key=key)


class StatusDetails(BaseModel):
    """Details about the status of a device."""

    rack: int = 0
    """The key of the rack the device is connected to."""
    device: str = ""
    """The key of the device."""


Status = status.Status[StatusDetails]
"""The status of a device."""


class Device(BaseModel):
    key: str = ""
    location: str = ""
    rack: int = 0
    name: str = ""
    make: str = ""
    model: str = ""
    configured: bool = False
    properties: str = ""
    status: Status | None = None

    @property
    def ontology_id(self) -> ontology.ID:
        """Returns the ontology ID for this Device."""
        return ontology_id(self.key)
