#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import Payload

from synnax.ontology import ID

DEVICE_ONTOLOGY_TYPE = ID(type="device")


def ontology_id(key: str) -> ID:
    """Returns the ontology ID for the Device entity."""
    return ID(type=DEVICE_ONTOLOGY_TYPE.type, key=key)


class Device(Payload):
    key: str = ""
    location: str = ""
    rack: int = 0
    name: str = ""
    make: str = ""
    model: str = ""
    configured: bool = False
    properties: str = ""

    @property
    def ontology_id(self) -> ID:
        """Returns the ontology ID for this Device."""
        return ontology_id(self.key)
