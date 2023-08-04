#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Literal

from dataclasses import dataclass

from freighter import Payload

from synnax.telem import DataType, Rate
from synnax.util.normalize import normalize

ChannelKey = int
ChannelName = str
ChannelKeys = tuple[int] | list[int]
ChannelNames = tuple[str] | list[str]
ChannelParams = ChannelKeys | ChannelNames | ChannelKey | ChannelName


class ChannelPayload(Payload):
    """A payload container that represent the properties of a channel exchanged to and
    from the Synnax server.
    """

    key: ChannelKey = 0
    data_type: DataType
    rate: Rate = Rate(0)
    name: str = ""
    leaseholder: int = 0
    is_index: bool = False
    index: ChannelKey = 0

    def __str__(self):
        return f"Channel {self.name}"


@dataclass
class NormalizedChannelParams:
    single: bool
    variant: Literal["keys", "names"]
    params: ChannelNames | ChannelKeys


def normalize_channel_params(
    params: ChannelParams,
) -> NormalizedChannelParams:
    """Determine if a list of keys or names is a single key or name."""
    normalized = normalize(params)
    if len(normalized) == 0:
        raise ValueError("no keys or names provided")
    return NormalizedChannelParams(
        single=isinstance(params, (str, int)),
        variant="keys" if isinstance(normalized[0], int) else "names",
        params=normalized,
    )
