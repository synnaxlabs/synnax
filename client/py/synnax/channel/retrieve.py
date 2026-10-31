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
from uuid import UUID

from pydantic import BaseModel

from alamos import NOOP, Instrumentation, trace
from freighter import UnaryClient
from synnax.channel.payload import (
    Key,
    NormalizedNameResult,
    Params,
    Payload,
    normalize_params,
)
from synnax.channel.types_gen import RetrieveRequest, RetrieveResponse, Where
from synnax.exceptions import NotFoundError
from synnax.util import one
from x import filter


class _AliasedRetrieveRequest(BaseModel):
    request: RetrieveRequest
    range_key: UUID | None = None


def params_where(channels: Params) -> Where:
    """Returns a filter that selects the channels that channels names by key or
    name.
    """
    normal = normalize_params(channels)
    if isinstance(normal, NormalizedNameResult):
        return Where(name_patterns=normal.channels)
    return Where(key=filter.Set[Key](in_=normal.channels))


def single_where(param: Key | str) -> Where:
    """Returns a filter that selects the channel that param names by key, or by a name
    that the Core reads as a pattern. A numeric string is a key.
    """
    normal = normalize_params(param)
    if isinstance(normal, NormalizedNameResult):
        return Where(name_patterns=normal.channels)
    return Where(key=filter.Set[Key](eq=normal.channels[0]))


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
        if len(normalize_params(channels).channels) == 0:
            return []
        return self.filter(RetrieveRequest(where=params_where(channels)))

    @trace("debug")
    def retrieve_one(self, param: Key | str) -> Payload:
        """Returns the one channel that param names.

        :raises NotFoundError: If no channel matches param.
        :raises MultipleFoundError: If more than one channel matches param.
        """
        return one("channel", single_where(param), self.retrieve(param))

    @trace("debug")
    def filter(
        self, req: RetrieveRequest, range_key: UUID | None = None
    ) -> list[Payload]:
        """Returns the channels req selects. A range_key sets each channel's alias in
        the range, and a search term then also matches those aliases, ranked first.
        """
        return self._client.send(
            "/channel/retrieve",
            _AliasedRetrieveRequest(request=req, range_key=range_key),
            RetrieveResponse,
        ).channels


def retrieve_required(r: Retriever, channels: Params) -> list[Payload]:
    normal = normalize_params(channels)
    results = r.retrieve(channels)
    found: set[Key | str]
    if isinstance(normal, NormalizedNameResult):
        found = {c.name for c in results}
    else:
        found = {c.key for c in results}
    not_found: list[Key | str] = [p for p in normal.channels if p not in found]
    if len(not_found) > 0:
        raise NotFoundError(f"channels not found matching {json.dumps(not_found)}")
    return results
