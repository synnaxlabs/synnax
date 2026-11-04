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
from synnax.resource import ID, CrudeID, Type


class _CreateReq(BaseModel):
    key: UUID | None = None
    name: str
    scope: Type | None = None
    parent: ID | None = None


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


class _SetParentReq(BaseModel):
    keys: list[UUID]
    parent: ID | None = None


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient):
        self._client = client

    def create(
        self,
        name: str,
        *,
        scope: Type | None = None,
        parent: CrudeID | None = None,
        key: str | None = None,
    ) -> Group:
        """Creates a group, or updates the group with the given key.

        :param name: The name of the group.
        :param scope: The type of resource the group holds. Required when the group
            has no parent.
        :param parent: The group, project, or rack to nest the group under. The group
            takes its scope from the parent.
        :param key: The key of the group to update.
        :returns: The created group.
        """
        return self._client.send(
            "/group/create",
            _CreateReq(
                key=UUID(key) if key else None,
                name=name,
                scope=scope,
                parent=ID(parent) if parent is not None else None,
            ),
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
            "/group/retrieve", _RetrieveReq(keys=keys), _RetrieveRes
        )
        if not is_single:
            return res.groups
        if len(res.groups) == 0:
            raise NotFoundError(f"Group matching {key} not found")
        return res.groups[0]

    def rename(self, key: UUID, name: str) -> Empty:
        return self._client.send("/group/rename", _RenameReq(key=key, name=name), Empty)

    def delete(self, keys: list[UUID]) -> Empty:
        return self._client.send("/group/delete", _DeleteReq(keys=keys), Empty)

    def set_parent(self, keys: list[UUID], parent: CrudeID | None = None) -> Empty:
        """Moves groups under a parent, or to the top level when the parent is None.

        :param keys: The keys of the groups to move.
        :param parent: The group, project, or rack to move the groups under.
        """
        return self._client.send(
            "/group/set-parent",
            _SetParentReq(keys=keys, parent=ID(parent) if parent is not None else None),
            Empty,
        )
