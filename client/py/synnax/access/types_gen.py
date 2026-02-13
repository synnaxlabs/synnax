#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from __future__ import annotations

from uuid import UUID

from pydantic import BaseModel

from synnax import ontology

ONTOLOGY_TYPE = ontology.ID(type="role")


class Role(BaseModel):
    key: UUID | None = None
    name: str
    description: str = ""
    internal: bool = False


def ontology_id(key: UUID | None = None) -> ontology.ID:
    return ontology.ID(type="role", key=key if key is None else str(key))
