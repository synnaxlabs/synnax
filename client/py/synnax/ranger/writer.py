#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from pydantic import BaseModel

from alamos import NOOP, Instrumentation, trace
from freighter import Empty, UnaryClient
from synnax.ontology.payload import ID
from synnax.ranger.payload import Key, Payload


class _ParentRef(BaseModel):
    key: UUID


class _CreatePayload(Payload):
    parent: _ParentRef | None = None


class _CreateRequest(BaseModel):
    ranges: list[_CreatePayload] = []


class _CreateResponse(BaseModel):
    ranges: list[Payload] = []


class _DeleteRequest(BaseModel):
    keys: list[Key]


class Writer:
    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @trace("debug", "range.create")
    def create(
        self, ranges: list[Payload], *, parent: ID | None = None
    ) -> list[Payload]:
        parent_ref = _ParentRef(key=UUID(parent.key)) if parent is not None else None
        wire = [_CreatePayload(**r.model_dump(), parent=parent_ref) for r in ranges]
        req = _CreateRequest(ranges=wire)
        return self._client.send("/range/create", req, _CreateResponse).ranges

    @trace("debug", "range.delete")
    def delete(self, keys: list[Key]) -> None:
        req = _DeleteRequest(keys=keys)
        self._client.send("/range/delete", req, Empty)
