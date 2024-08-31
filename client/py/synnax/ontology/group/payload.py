#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from freighter import Payload
from uuid import UUID
from synnax.ontology.payload import ID


class Group(Payload):
    key: UUID
    name: str

    @property
    def ontology_id(self) -> ID:
        return ID(key=str(self.key), type="group")
