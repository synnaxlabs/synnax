#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal, Sequence, TypeAlias, cast

from pydantic import BaseModel

from synnax import ontology
from synnax.telem import DataType, TimeSpan
from synnax.util.normalize import normalize

Key = int

ONTOLOGY_TYPE = ontology.ID(type="channel")


def ontology_id(key: Key) -> ontology.ID:
    """Returns the ontology ID for the Channel entity."""
    return ontology.ID(type=ONTOLOGY_TYPE.type, key=str(key))


OPERATION_TYPES = Literal["min", "max", "avg", "none"]


class Operation(BaseModel):
    """Represents an operation on a calculated channel."""

    type: OPERATION_TYPES
    reset_channel: Key = 0
    duration: TimeSpan = TimeSpan(0)


class Payload(BaseModel):
    """A payload container that represent the properties of a channel exchanged to and
    from the Synnax server.
    """

    key: Key = 0
    data_type: DataType
    name: str = ""
    leaseholder: int = 0
    is_index: bool = False
    index: Key = 0
    internal: bool = False
    virtual: bool = False
    expression: str | None = ""
    operations: list[Operation] | None = None

    def __str__(self):
        return f"Channel(name={self.name}, key={self.key})"

    def __hash__(self) -> int:
        return hash(self.key)


@dataclass
class NormalizedChannelKeyResult:
    single: bool
    variant: Literal["keys"]
    channels: list[Key] | tuple[Key]


@dataclass
class NormalizedChannelNameResult:
    single: bool
    variant: Literal["names"]
    channels: list[str]


def normalize_params(
    channels: Params,
) -> NormalizedChannelKeyResult | NormalizedChannelNameResult:
    """Determine if a list of keys or names is a single key or name."""
    normalized = normalize(channels)
    if len(normalized) == 0:
        return NormalizedChannelKeyResult(single=False, variant="keys", channels=[])
    single = isinstance(channels, (Key, str))
    if isinstance(normalized[0], str):
        try:
            str_list = cast(list[str], normalized)
            numeric_strings = [Key(s) for s in str_list]
            return NormalizedChannelKeyResult(
                single=single,
                variant="keys",
                channels=numeric_strings,
            )
        except ValueError:
            return NormalizedChannelNameResult(
                single=single,
                variant="names",
                channels=cast(list[str], normalized),
            )
    elif isinstance(normalized[0], Payload):
        payload_list = cast(list[Payload], normalized)
        return NormalizedChannelKeyResult(
            single=single,
            variant="keys",
            channels=[c.key for c in payload_list],
        )
    return NormalizedChannelKeyResult(
        single=single,
        variant="keys",
        channels=normalized,
    )


Params: TypeAlias = (
    Key | Sequence[Key] | str | Sequence[str] | Sequence[Payload] | Payload
)


def has_params(channels: Params | None) -> bool:
    if channels is None:
        return False
    if isinstance(channels, (Key, str, Payload)):
        return True
    return len(channels) > 0
