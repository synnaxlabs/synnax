#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations
from freighter import Payload


class ID(Payload):
    key: str | None = ""
    type: str | None = ""

    def __init__(self, key: CrudeID | None = None, type: str | None = None):
        if isinstance(key, ID):
            super().__init__(key=key.key, type=key.type)
        elif isinstance(key, tuple):
            key, type = key
            super().__init__(key=key, type=type)
        elif type is None:
            type, key = key.split(":")
            super().__init__(key=key, type=type)
        else:
            super().__init__(key=key, type=type)


ROOT = ID(key="root", type="builtin")

CrudeID = str | ID


class Resource(Payload):
    id: ID
    name: str
    data: dict


class Relationship(Payload):
    from_: ID
    type: str
    to: ID
