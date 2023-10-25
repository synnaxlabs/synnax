#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass
from typing import Literal
from uuid import UUID

from freighter import Payload

from synnax.exceptions import QueryError
from synnax.telem import TimeRange
from synnax.util.normalize import normalize


class RangePayload(Payload):
    """Network transportable payload representing a range."""

    key: UUID = UUID(int=0)
    name: str = ""
    time_range: TimeRange


RangeKey = UUID
"""The type for the key of a Range. A UUID."""
RangeName = str
"""The type for the name of a Range. A string."""
RangeKeys = tuple[UUID] | list[UUID]
"""The type for the keys of a Range. A tuple or list of UUIDs."""
RangeNames = tuple[str] | list[str]
"""The type for the names of a Range. A tuple or list of strings."""
RangeParams = RangeKeys | RangeNames | RangeKey | RangeName
"""Parameters that can be used to query a range"""


@dataclass
class NormalizeRangeParams:
    single: bool
    variant: Literal["keys", "names"]
    params: RangeNames | RangeKeys


def normalize_range_params(
    params: RangeParams,
) -> NormalizeRangeParams:
    normalized = normalize(params)
    if len(normalized) == 0:
        raise QueryError("No keys or names provided to range retrieval")
    return NormalizeRangeParams(
        single=isinstance(params, (str, UUID)),
        variant="keys" if isinstance(normalized[0], UUID) else "names",
        params=normalized,
    )
