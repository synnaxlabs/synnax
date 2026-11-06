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
from synnax.user.types_gen import RetrieveRequest, Where
from x import filter
from x.lists import normalize
from x.params import require_named_params


class _CreateRequest(BaseModel):
    users: list[New]


class _CreateResponse(BaseModel):
    users: list[User]


class _RetrieveResponse(BaseModel):
    users: list[User] = Field(default_factory=list)


class _DeleteRequest(BaseModel):
    keys: list[UUID]


class _ChangeUsernameRequest(BaseModel):
    key: UUID
    username: str


class _ChangeNameRequest(BaseModel):
    key: UUID
    name: str


class Client:
    client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        self.client = transport

    @overload
    def create(
        self,
        *,
        username: str,
        password: str,
        name: str = "",
        key: UUID | None = None,
    ) -> User: ...

    @overload
    def create(self, *, user: New) -> User: ...

    @overload
    def create(self, *, users: list[New]) -> list[User]: ...

    @require_named_params(example_params=("user", "New(username='synnax')"))
    def create(
        self,
        *,
        username: str | None = None,
        password: str | None = None,
        name: str = "",
        key: UUID | None = None,
        user: New | None = None,
        users: list[New] | None = None,
    ) -> User | list[User]:
        if username is not None:
            if password is None:
                raise ValueError("password is required when username is provided")
            user = New(
                username=username,
                password=password,
                name=name,
                key=key,
            )
        single = user is not None
        if user is not None:
            users = [user]
        if users is None:
            raise ValueError("Either username, user, or users must be provided")
        res = self.client.send(
            "/user/create", _CreateRequest(users=users), _CreateResponse
        ).users
        if single:
            return res[0]
        return res

    def change_username(self, key: UUID, username: str) -> None:
        self.client.send(
            "/user/change-username",
            _ChangeUsernameRequest(key=key, username=username),
            Empty,
        )

    def change_name(self, key: UUID, *, name: str) -> None:
        self.client.send("/user/rename", _ChangeNameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(self, *, key: UUID) -> User: ...

    @overload
    def retrieve(self, *, username: str) -> User: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[UUID] | None = None,
        usernames: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[User]: ...

    @require_named_params(example_params=("username", "synnax"))
    def retrieve(
        self,
        *,
        key: UUID | None = None,
        username: str | None = None,
        where: Where | None = None,
        keys: list[UUID] | None = None,
        usernames: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> User | list[User]:
        """Retrieves users. Returns one user when a key or username is given, and a list
        otherwise.

        :param where: Keeps users that match this filter tree.
        :param search_term: A fuzzy search term matched against user names.
        :raises NotFoundError: If a single user is requested and none matches.
        """
        if key is not None:
            keys = [key]
        if username is not None:
            usernames = [username]
        flat = Where(
            key=None if keys is None else filter.Set[UUID](in_=keys),
            usernames=usernames,
        )
        single = key is not None or username is not None
        res = self.client.send(
            "/user/retrieve",
            RetrieveRequest(
                where=filter.merge(where, flat),
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            _RetrieveResponse,
        )
        users = res.users
        if not single:
            return users
        if len(users) == 0:
            raise NotFoundError(f"User matching {key or username} not found")
        return users[0]

    def delete(self, keys: UUID | list[UUID] | None = None) -> None:
        self.client.send("/user/delete", _DeleteRequest(keys=normalize(keys)), Empty)
