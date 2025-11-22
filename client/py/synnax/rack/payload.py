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

RACK_ONTOLOGY_TYPE = ID(type="rack")


def ontology_id(key: int) -> ID:
    """Returns the ontology ID for the Rack entity."""
    return ID(type=RACK_ONTOLOGY_TYPE.type, key=key)


class Rack(Payload):
    key: int = 0
    name: str = ""
    task_counter: int = 0
    embedded: bool = False

    @property
    def ontology_id(self) -> ID:
        """Returns the ontology ID for this Rack."""
        return ontology_id(self.key)
