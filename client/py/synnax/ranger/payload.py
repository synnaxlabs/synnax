#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal

from dataclasses import dataclass

from uuid import UUID

from freighter import Payload

from synnax.telem import TimeRange
from synnax.util.normalize import normalize


class RangePayload(Payload):
    key: UUID = UUID(int=0)
    name: str = ""
    time_range: TimeRange


RangeKey = UUID
RangeName = str
RangeKeys = tuple[UUID] | list[UUID]
RangeNames = tuple[str] | list[str]
RangeParams = RangeKeys | RangeNames | RangeKey | RangeName


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
        raise ValueError("no keys or names provided")
    return NormalizeRangeParams(
        single=isinstance(params, (str, UUID)),
        variant="keys" if isinstance(normalized[0], UUID) else "names",
        params=normalized,
    )
