#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from pydantic import BaseModel, ConfigDict

from synnax import ontology
from synnax.color import Color
from synnax.telem import TimeRange

ONTOLOGY_TYPE = ontology.ID(type="range")


def ontology_id(key: UUID) -> ontology.ID:
    """Returns the ontology ID for the Range entity."""
    return ontology.ID(type=ONTOLOGY_TYPE.type, key=str(key))


class Payload(BaseModel):
    """Network transportable payload representing a range."""

    model_config = ConfigDict(validate_assignment=True)

    key: UUID = UUID(int=0)
    name: str = ""
    time_range: TimeRange
    color: Color = Color()


Key = UUID | str
"""The type for the names of a Range. A tuple or list of strings."""
RangeParams = Key | list[Key] | tuple[Key] | str | list[str] | tuple[str]
"""Parameters that can be used to query a range"""
