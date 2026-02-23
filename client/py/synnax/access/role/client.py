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

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient, send_required
from pydantic import BaseModel

from synnax.access.role.payload import Role
from synnax.util.normalize import normalize


class _CreateRequest(BaseModel):
    roles: list[Role]


_CreateResponse = _CreateRequest


class _RetrieveRequest(BaseModel):
    keys: list[UUID] | None = None
    limit: int | None = None
    offset: int | None = None
    internal: bool | None = None


class _RetrieveResponse(BaseModel):
    roles: list[Role] | None


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
        res = send_required(self._client, "/access/role/create", req, _CreateResponse)
        return res.roles[0] if is_single else res.roles

    @overload
    def retrieve(self, key: UUID) -> Role: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[UUID] | None = None,
        limit: int | None = None,
        offset: int | None = None,
        internal: bool | None = None,
    ) -> list[Role]: ...

    def retrieve(
        self,
        key: UUID | None = None,
        keys: list[UUID] | None = None,
        limit: int | None = None,
        offset: int | None = None,
        internal: bool | None = None,
    ) -> Role | list[Role]:
        is_single = key is not None
        if is_single and key is not None:
            keys = [key]
        req = _RetrieveRequest(keys=keys, limit=limit, offset=offset, internal=internal)
        res = send_required(
            self._client, "/access/role/retrieve", req, _RetrieveResponse
        )
        roles = [] if res.roles is None else res.roles
        return roles[0] if is_single else roles

    def delete(self, keys: UUID | list[UUID]) -> None:
        req = _DeleteRequest(keys=normalize(keys))
        send_required(self._client, "/access/role/delete", req, Empty)

    def assign(self, user: UUID, role: UUID) -> None:
        """Assign a role to a user.

        Args:
            user: The UUID of the user to assign the role to
            role: The UUID of the role to assign
        """
        req = _AssignRequest(user=user, role=role)
        send_required(self._client, "/access/role/assign", req, Empty)

    def unassign(self, user: UUID, role: UUID) -> None:
        """Remove a role from a user.

        Args:
            user: The UUID of the user to unassign the role from
            role: The UUID of the role to unassign
        """
        req = _UnassignRequest(user=user, role=role)
        send_required(self._client, "/access/role/unassign", req, Empty)
