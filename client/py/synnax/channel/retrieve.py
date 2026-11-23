#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import json
from dataclasses import dataclass

from alamos import NOOP, Instrumentation, trace
from freighter import UnaryClient
from synnax.channel.payload import (
    Key,
    Lookup,
    Params,
    Payload,
    lookup_where,
    nfc,
    split_lookups,
    to_lookups,
)
from synnax.channel.types_gen import RetrieveRequest, RetrieveResponse
from synnax.exceptions import NotFoundError, ValidationError
from synnax.util import sole


class Retriever:
    """Retrieves channel payloads from the Core."""

    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def retrieve(self, channels: Params) -> list[Payload]:
        """Returns every channel that channels names. A name can match several
        channels.
        """
        lookups = to_lookups(channels)
        if len(lookups) == 0:
            return []
        return self.filter(RetrieveRequest(where=lookup_where(lookups)))

    @trace("debug")
    def retrieve_one(self, param: Lookup) -> Payload:
        """Returns the one channel that param names.

        :raises NotFoundError: If no channel matches param.
        :raises MultipleFoundError: If several channels hold the name param.
        """
        lookups = to_lookups(param)
        req = RetrieveRequest(
            where=lookup_where(lookups), strict=isinstance(lookups[0], str)
        )
        return sole("channel", req, self.filter(req))

    @trace("debug")
    def filter(self, req: RetrieveRequest) -> list[Payload]:
        """Returns the channels req selects."""
        return self._client.send("/channel/retrieve", req, RetrieveResponse).channels


def _text(lookups: object) -> str:
    return json.dumps(lookups, ensure_ascii=False)


def not_found(lookups: list[Lookup]) -> NotFoundError:
    """Returns the error for lookups that find no channel."""
    return NotFoundError(f"channels not found matching {_text(lookups)}")


def named_twice(first: Lookup, second: Lookup) -> ValidationError:
    """Returns the error for two lookups that name one channel."""
    return ValidationError(f"{_text(first)} and {_text(second)} name one channel")


def refuse_shared(refs: list[Lookup], keys: list[Key]) -> None:
    """Checks that the entries of one write or record name different channels.

    :param keys: The key of the channel of each entry.
    :raises ValidationError: If two entries name one channel.
    """
    if len(set(keys)) == len(keys):
        return
    first: dict[Key, Lookup] = {}
    for ref, key in zip(refs, keys, strict=True):
        if key in first:
            raise named_twice(first[key], ref)
        first[key] = ref


def match(r: Retriever, channels: Params) -> list[Payload | None]:
    """Returns the channel of each lookup in channels, in order, from one strict
    request, or None for a lookup that finds no channel. A channel matches a lookup by
    its key, its name, or its path, in NFC.

    :raises MultipleFoundError: If several channels hold a name.
    """
    lookups = to_lookups(channels)
    if len(lookups) == 0:
        return []
    split = split_lookups(lookups)
    named = len(split.names) > 0 or len(split.paths) > 0
    req = RetrieveRequest(
        where=lookup_where(lookups), strict=named, include_path=len(split.paths) > 0
    )
    found: dict[Key | str, Payload] = {}
    for ch in r.filter(req):
        found[ch.key] = ch
        if named:
            found[ch.name] = ch
            if ch.path is not None:
                found[ch.path] = ch
    return [found.get(nfc(lookup)) for lookup in lookups]


def resolve(r: Retriever, channels: Params) -> list[Payload]:
    """Returns the channel of each lookup in channels, in order, from one strict
    request, as match does.

    :raises NotFoundError: Naming every lookup that finds no channel.
    :raises MultipleFoundError: If several channels hold a name.
    """
    lookups = to_lookups(channels)
    found = match(r, lookups)
    missing = [lk for lk, ch in zip(lookups, found, strict=True) if ch is None]
    if len(missing) > 0:
        raise not_found(missing)
    return [ch for ch in found if ch is not None]


@dataclass
class Resolved:
    """The channels of an open, a read, or an acquire."""

    lookups: list[Lookup]
    """The channels as the caller gave them, each payload as its key."""
    each: list[Payload]
    """The channel of each lookup, at the index of its lookup."""
    channels: list[Payload]
    """Each channel once, in the order of its first lookup."""


def resolve_open(r: Retriever, channels: Params) -> Resolved:
    """Resolves the channels of an open, a read, or an acquire in one strict request.
    Lookups compare in NFC, so a repeat of one lookup keeps one channel.

    :raises NotFoundError: Naming every lookup that finds no channel.
    :raises ValidationError: If two different lookups name one channel.
    :raises MultipleFoundError: If several channels hold a name.
    """
    lookups = to_lookups(channels)
    each = resolve(r, lookups)
    first: dict[Key, Lookup] = {}
    distinct: list[Payload] = []
    for lookup, ch in zip(lookups, each, strict=True):
        if ch.key not in first:
            first[ch.key] = lookup
            distinct.append(ch)
        elif nfc(first[ch.key]) != nfc(lookup):
            raise named_twice(first[ch.key], lookup)
    return Resolved(lookups=lookups, each=each, channels=distinct)
