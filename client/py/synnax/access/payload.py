#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from __future__ import annotations
from uuid import UUID
from freighter import Payload
from synnax.ontology.id import OntologyID


class Policy(Payload):
    key: UUID = UUID(int=0)
    subjects: list[OntologyID] = None
    objects: list[OntologyID] = None
    actions: list[str] = None

    def __str__(self):
        return f"Policy with subjects {self.subjects}, objects {self.objects}, actions {self.actions}"

    def __hash__(self) -> int:
        return hash(self.key)
