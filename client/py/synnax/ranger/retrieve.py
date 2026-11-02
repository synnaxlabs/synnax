#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel, Field

from alamos import NOOP, Instrumentation, trace
from freighter import UnaryClient
from synnax import label
from synnax.ranger.types_gen import Key, Payload
from x.telem import TimeRange


class Request(BaseModel):
    """The filters for a range retrieval. A field left as None does not filter."""

    keys: list[Key] | None = None
    names: list[str] | None = None
    search_term: str | None = None
    overlaps_with: TimeRange | None = None
    has_labels: list[label.Key] | None = None
    limit: int | None = None
    offset: int | None = None
    include_labels: bool | None = None
    include_parent: bool | None = None


class _Response(BaseModel):
    ranges: list[Payload] = Field(default_factory=list)


class Retriever:
    _client: UnaryClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @trace("debug")
    def retrieve(self, req: Request) -> list[Payload]:
        return self._client.send("/range/retrieve", req, _Response).ranges
