#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import re
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
from x import filter

_NAME_LITERAL = re.compile(r"[a-zA-Z_][a-zA-Z0-9_]*")


class _RetrieveRequest(BaseModel):
    request: RetrieveRequest
    range_key: UUID | None = None


def names_where(names: list[str]) -> Where:
    """Returns a filter that selects channels by name. A name that is not a valid
    channel name is a regular expression, anchored when unanchored.
    """
    literals = [n for n in names if _NAME_LITERAL.fullmatch(n)]
    lines = [
        Where(name=filter.String(regex=_anchor(n)))
        for n in names
        if not _NAME_LITERAL.fullmatch(n)
    ]
    if len(literals) > 0:
        lines.insert(0, Where(name=filter.String(in_=literals)))
    return lines[0] if len(lines) == 1 else Where(or_=lines)


def _anchor(pattern: str) -> str:
    if pattern.startswith("^") or pattern.endswith("$"):
        return pattern
    return f"^{pattern}$"


def params_where(channels: Params) -> Where:
    """Returns a filter that selects the channels that channels names by key or
    name.
    """
    normal = normalize_params(channels)
    if isinstance(normal, NormalizedNameResult):
        return names_where(normal.channels)
    return Where(key=filter.Set[Key](in_=normal.channels))


class Retriever:
    """Retrieves channel payloads from the Core."""

    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
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
        channels = self.retrieve(param)
        if len(channels) == 0:
            raise NotFoundError(f"Could not find channel matching {param}")
        return channels[0]

    @trace("debug")
    def filter(
        self, req: RetrieveRequest, range_key: UUID | None = None
    ) -> list[Payload]:
        """Returns the channels req selects. A range_key sets each channel's alias in
        the range, and a search term then also matches those aliases, ranked first.
        """
        return self._client.send(
            "/channel/retrieve",
            _RetrieveRequest(request=req, range_key=range_key),
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
        raise NotFoundError(f"Could not find channels: {not_found}")
    return results
