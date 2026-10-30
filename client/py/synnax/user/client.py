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

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax.exceptions import NotFoundError
from synnax.user.payload import New, User
from x.lists import normalize
from x.params import require_named_params


class _CreateRequest(BaseModel):
    users: list[New]


class _CreateResponse(BaseModel):
    users: list[User]


class _RetrieveRequest(BaseModel):
    keys: list[UUID] | None = None


class _RetrieveResponse(BaseModel):
    users: list[User] = Field(default_factory=list)


class _DeleteRequest(BaseModel):
    keys: list[UUID]


class _ChangeNameRequest(BaseModel):
    key: UUID
    name: str


class Client:
    client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        self.client = transport

    @overload
    def create(self, *, name: str, key: UUID | None = None) -> User: ...

    @overload
    def create(self, *, user: New) -> User: ...

    @overload
    def create(self, *, users: list[New]) -> list[User]: ...

    @require_named_params(example_params=("user", "New(name='Hari Seldon')"))
    def create(
        self,
        *,
        name: str | None = None,
        key: UUID | None = None,
        user: New | None = None,
        users: list[New] | None = None,
    ) -> User | list[User]:
        """Creates users with no password. Give a user its first password with
        client.auth.passwords.create.
        """
        if name is not None:
            user = New(name=name, key=key)
        single = user is not None
        if user is not None:
            users = [user]
        if users is None:
            raise ValueError("Either name, user, or users must be provided")
        res = self.client.send(
            "/user/create", _CreateRequest(users=users), _CreateResponse
        ).users
        if single:
            return res[0]
        return res

    def change_name(self, key: UUID, *, name: str) -> None:
        self.client.send("/user/rename", _ChangeNameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(self, *, key: UUID) -> User: ...

    @overload
    def retrieve(self, *, keys: list[UUID]) -> list[User]: ...

    @require_named_params(example_params=("key", "UUID('...')"))
    def retrieve(
        self,
        *,
        key: UUID | None = None,
        keys: list[UUID] | None = None,
    ) -> User | list[User]:
        if key is not None:
            keys = normalize(key)
        res = self.client.send(
            "/user/retrieve", _RetrieveRequest(keys=keys), _RetrieveResponse
        )
        users = res.users
        if key is None:
            return users
        if len(users) == 0:
            raise NotFoundError(f"User {key} not found")
        return users[0]

    def delete(self, keys: UUID | list[UUID] | None = None) -> None:
        self.client.send("/user/delete", _DeleteRequest(keys=normalize(keys)), Empty)
