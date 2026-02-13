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
from typing import Literal, Sequence, TypeAlias

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


@dataclass
class NormalizedKeyResult:
    single: bool
    variant: Literal["keys"]
    channels: list[Key] | tuple[Key]


@dataclass
class NormalizedNameResult:
    single: bool
    variant: Literal["names"]
    channels: list[str]


def normalize_params(
    channels: Params,
) -> NormalizedKeyResult | NormalizedNameResult:
    """Determine if a list of keys or names is a single key or name."""
    normalized = normalize(channels)
    if len(normalized) == 0:
        return NormalizedKeyResult(single=False, variant="keys", channels=[])
    single = isinstance(channels, (Key, str))
    if isinstance(normalized[0], str):
        str_list = [s for s in normalized if isinstance(s, str)]
        try:
            return NormalizedKeyResult(
                single=single,
                variant="keys",
                channels=[Key(s) for s in str_list],
            )
        except (ValueError, TypeError):
            return NormalizedNameResult(
                single=single,
                variant="names",
                channels=str_list,
            )
    elif isinstance(normalized[0], Payload):
        return NormalizedKeyResult(
            single=single,
            variant="keys",
            channels=[c.key for c in normalized if isinstance(c, Payload)],
        )
    return NormalizedKeyResult(
        single=single,
        variant="keys",
        channels=[k for k in normalized if isinstance(k, int)],
    )


Params: TypeAlias = (
    Key
    | str
    | Payload
    | Sequence[Key]
    | Sequence[str]
    | Sequence[Key | str]
    | Sequence[Key | str | Payload]
    | Sequence[Payload]
)


def has_params(channels: Params | None) -> bool:
    if channels is None:
        return False
    if isinstance(channels, (Key, str, Payload)):
        return True
    return len(channels) > 0
