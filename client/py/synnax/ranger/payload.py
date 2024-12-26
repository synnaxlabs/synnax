#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID
from freighter import Payload
from synnax.telem import TimeRange


class RangePayload(Payload):
    """Network transportable payload representing a range."""

    key: UUID = UUID(int=0)
    name: str = ""
    time_range: TimeRange
    color: str = "#000000"


RangeKey = UUID | str
"""The type for the key of a Range. A UUID."""
RangeName = str
"""The type for the name of a Range. A string."""
RangeKeys = tuple[UUID] | list[UUID]
"""The type for the keys of a Range. A tuple or list of UUIDs."""
RangeNames = tuple[str] | list[str]
"""The type for the names of a Range. A tuple or list of strings."""
RangeParams = RangeKeys | RangeNames | RangeKey | RangeName
"""Parameters that can be used to query a range"""
