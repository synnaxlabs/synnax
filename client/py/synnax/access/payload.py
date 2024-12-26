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
from synnax.ontology.payload import ID

ALLOW_ALL = ID(type="allow_all", key="")


class Policy(Payload):
    key: UUID | None = None
    subjects: list[ID] = None
    objects: list[ID] = None
    actions: list[str] = None
