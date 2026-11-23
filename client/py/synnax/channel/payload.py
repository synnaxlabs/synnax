#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import unicodedata
from collections.abc import Sequence
from dataclasses import dataclass, field
from typing import TypeAlias, TypeIs

from synnax import resource
from synnax.channel.types_gen import (
    Key as Key,
    Operation as Operation,
    Payload as Payload,
    Where,
)
from synnax.exceptions import ValidationError
from x import filter
from x.lists import normalize

_RESOURCE_TYPE = resource.ID(type="channel")


def _resource_id(key: Key) -> resource.ID:
    return resource.ID(type=_RESOURCE_TYPE.type, key=str(key))


Lookup: TypeAlias = Key | str
"""A channel named by its key, its name, or its path."""

Params: TypeAlias = Lookup | Payload | Sequence[Lookup | Payload]

_PATH_SEPARATOR = "."
_MAX_KEY = 4294967295


def to_lookups(params: Params) -> list[Lookup]:
    """Returns each channel of params as a lookup, a payload as its key.

    :raises ValidationError: If an integer does not fit a channel key.
    """
    lookups: list[Lookup] = []
    for p in normalize(params):
        if isinstance(p, Payload):
            lookups.append(p.key)
        elif isinstance(p, str):
            lookups.append(p)
        elif isinstance(p, int):
            if not 0 <= p <= _MAX_KEY:
                raise ValidationError(f"{p} is not a channel key")
            lookups.append(p)
        else:
            raise TypeError(f"{p!r} is not a channel key, name, path, or payload")
    return lookups


def nfc(lookup: Lookup) -> Lookup:
    """Returns lookup in NFC, the form in which lookups compare."""
    return unicodedata.normalize("NFC", lookup) if isinstance(lookup, str) else lookup


def is_path(text: str) -> bool:
    """Returns whether text names a channel by its path."""
    return _PATH_SEPARATOR in text


def is_single(params: Params) -> TypeIs[Key | str]:
    """Returns whether params names one channel by key, name, or path."""
    return isinstance(params, (Key, str))


@dataclass
class SplitLookups:
    """The lookups of each kind, in the order given."""

    keys: list[Key] = field(default_factory=list)
    names: list[str] = field(default_factory=list)
    paths: list[str] = field(default_factory=list)


def split_lookups(lookups: list[Lookup]) -> SplitLookups:
    """Splits lookups by kind. An integer is a key, a string that holds a '.' is a
    path, and any other string is a name, digits included.
    """
    split = SplitLookups()
    for lookup in lookups:
        if isinstance(lookup, int):
            split.keys.append(lookup)
        elif is_path(lookup):
            split.paths.append(lookup)
        else:
            split.names.append(lookup)
    return split


def lookup_where(lookups: list[Lookup]) -> Where:
    """Returns the filter that looks up each channel in lookups. Lookups of several
    kinds join in an or. An empty list selects no channel.
    """
    split = split_lookups(lookups)
    branches: list[Where] = []
    if split.keys or not lookups:
        keys = split.keys
        branches.append(
            Where(key=filter.Set[Key](eq=keys[0]))
            if len(keys) == 1
            else Where(key=filter.Set[Key](in_=keys))
        )
    if split.names:
        names = split.names
        branches.append(
            Where(name=filter.String(eq=names[0]))
            if len(names) == 1
            else Where(name=filter.String(in_=names))
        )
    if split.paths:
        paths = split.paths
        branches.append(
            Where(path=filter.Set[str](eq=paths[0]))
            if len(paths) == 1
            else Where(path=filter.Set[str](in_=paths))
        )
    return branches[0] if len(branches) == 1 else Where(or_=branches)


def has_params(channels: Params | None) -> bool:
    if channels is None:
        return False
    if isinstance(channels, (Key, str, Payload)):
        return True
    return len(channels) > 0
