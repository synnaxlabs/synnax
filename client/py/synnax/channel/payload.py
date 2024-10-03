#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal, cast

from freighter import Payload

from synnax.telem import DataType, Rate
from synnax.util.normalize import normalize

ChannelKey = int
ChannelName = str
ChannelKeys = list[int]
ChannelNames = list[str]
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
    internal: bool = False
    virtual: bool = False

    def __str__(self):
        return f"Channel(name={self.name}, key={self.key})"

    def __hash__(self) -> int:
        return hash(self.key)


@dataclass
class NormalizedChannelKeyResult:
    single: bool
    variant: Literal["keys"]
    channels: ChannelKeys


@dataclass
class NormalizedChannelNameResult:
    single: bool
    variant: Literal["names"]
    channels: ChannelNames


def normalize_channel_params(
    channels: ChannelParams,
) -> NormalizedChannelKeyResult | NormalizedChannelNameResult:
    """Determine if a list of keys or names is a single key or name."""
    normalized = normalize(channels)
    if len(normalized) == 0:
        return NormalizedChannelKeyResult(single=False, variant="keys", channels=[])
    single = isinstance(channels, (ChannelKey, ChannelName))
    if isinstance(normalized[0], str):
        try:
            numeric_strings = [ChannelKey(s) for s in normalized]
            return NormalizedChannelKeyResult(
                single=single,
                variant="keys",
                channels=cast(ChannelKeys, numeric_strings),
            )
        except ValueError:
            return NormalizedChannelNameResult(
                single=single,
                variant="names",
                channels=cast(ChannelNames, normalized),
            )
    elif isinstance(normalized[0], ChannelPayload):
        return NormalizedChannelNameResult(
            single=single,
            variant="keys",
            channels=cast(ChannelNames, [c.key for c in normalized]),
        )
    return NormalizedChannelKeyResult(
        single=single,
        variant="keys",
        channels=cast(ChannelKeys, normalized),
    )
