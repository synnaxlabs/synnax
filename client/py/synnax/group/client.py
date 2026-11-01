#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload
from uuid import UUID

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.exceptions import NotFoundError
from synnax.group.types_gen import Group
from synnax.ontology.payload import ID, CrudeID


class _CreateReq(BaseModel):
    parent: ID
    key: UUID | None = None
    name: str


class _RetrieveReq(BaseModel):
    keys: list[UUID] | None = None


class _RetrieveRes(BaseModel):
    groups: list[Group] = []


class _CreateRes(BaseModel):
    group: Group


class _RenameReq(BaseModel):
    key: UUID
    name: str


class _DeleteReq(BaseModel):
    keys: list[UUID]


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient):
        self._client = client

    def create(self, parent: CrudeID, name: str, *, key: str | None = None) -> Group:
        return self._client.send(
            "/ontology/create-group",
            _CreateReq(parent=ID(parent), key=UUID(key) if key else None, name=name),
            _CreateRes,
        ).group

    @overload
    def retrieve(self, key: UUID) -> Group: ...

    @overload
    def retrieve(self, *, keys: list[UUID]) -> list[Group]: ...

    def retrieve(
        self,
        key: UUID | None = None,
        *,
        keys: list[UUID] | None = None,
    ) -> Group | list[Group]:
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = self._client.send(
            "/ontology/retrieve-group", _RetrieveReq(keys=keys), _RetrieveRes
        )
        if not is_single:
            return res.groups
        if len(res.groups) == 0:
            raise NotFoundError(f"Group matching {key} not found")
        return res.groups[0]

    def rename(self, key: UUID, name: str) -> Empty:
        return self._client.send(
            "/ontology/rename-group", _RenameReq(key=key, name=name), Empty
        )

    def delete(self, keys: list[UUID]) -> Empty:
        return self._client.send("/ontology/delete-group", _DeleteReq(keys=keys), Empty)
