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
import unicodedata

from alamos import NOOP, Instrumentation, trace
from freighter import UnaryClient
from synnax.channel.payload import (
    Key,
    Lookup,
    Params,
    Payload,
    lookup_where,
    split_lookups,
    to_lookups,
)
from synnax.channel.types_gen import RetrieveRequest, RetrieveResponse
from synnax.exceptions import NotFoundError
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


def _nfc(lookup: Lookup) -> Lookup:
    return unicodedata.normalize("NFC", lookup) if isinstance(lookup, str) else lookup


def match(r: Retriever, channels: Params) -> list[Payload | None]:
    """Returns the channel of each lookup in channels, in order, from one strict
    request, or None for a lookup that finds no channel. A channel matches a lookup by
    its key, its name in NFC, or its path in NFC.

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
            found[_nfc(ch.name)] = ch
            if ch.path is not None:
                found[_nfc(ch.path)] = ch
    return [found.get(_nfc(lookup)) for lookup in lookups]


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
        raise NotFoundError(f"channels not found matching {json.dumps(missing)}")
    return [ch for ch in found if ch is not None]
