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

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.access.role.types_gen import (
    RetrieveRequest,
    RetrieveResponse,
    Role,
    Where,
)
from synnax.util import one
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    roles: list[Role]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[UUID]


class _AssignRequest(BaseModel):
    user: UUID
    role: UUID


class _UnassignRequest(BaseModel):
    user: UUID
    role: UUID


class Client:
    _client: UnaryClient
    instrumentation: Instrumentation

    def __init__(
        self,
        *,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ):
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        roles: Role,
    ) -> Role: ...

    @overload
    def create(
        self,
        roles: list[Role],
    ) -> list[Role]: ...

    def create(
        self,
        roles: Role | list[Role],
    ) -> Role | list[Role]:
        is_single = not isinstance(roles, list)
        req = _CreateRequest(roles=normalize(roles))
        res = self._client.send("/access/role/create", req, _CreateResponse)
        return res.roles[0] if is_single else res.roles

    @overload
    def retrieve(self, key: UUID) -> Role: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[UUID] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        internal: bool | None = None,
    ) -> list[Role]: ...

    def retrieve(
        self,
        key: UUID | None = None,
        *,
        where: Where | None = None,
        keys: list[UUID] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        internal: bool | None = None,
    ) -> Role | list[Role]:
        """Retrieves roles. Returns one role when a key is given, and a list otherwise.

        :param where: Keeps roles that match this filter tree.
        :raises NotFoundError: If a single role is requested and none matches.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        flat = Where(
            key=None if keys is None else filter.Set[UUID](in_=keys),
            internal=None if internal is None else filter.Bool(eq=internal),
        )
        req = RetrieveRequest(
            where=filter.merge(where, flat),
            search_term=search_term,
            limit=limit,
            offset=offset,
        )
        res = self._client.send("/access/role/retrieve", req, RetrieveResponse)
        if not is_single:
            return res.roles
        return one("Role", key, res.roles)

    def delete(self, keys: UUID | list[UUID]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        self._client.send("/access/role/delete", req, Empty)

    def assign(self, *, user: UUID, role: UUID) -> None:
        """Assign a role to a user.

        Args:
            user: The UUID of the user to assign the role to
            role: The UUID of the role to assign
        """
        req = _AssignRequest(user=user, role=role)
        self._client.send("/access/role/assign", req, Empty)

    def unassign(self, *, user: UUID, role: UUID) -> None:
        """Remove a role from a user.

        Args:
            user: The UUID of the user to unassign the role from
            role: The UUID of the role to unassign
        """
        req = _UnassignRequest(user=user, role=role)
        self._client.send("/access/role/unassign", req, Empty)
