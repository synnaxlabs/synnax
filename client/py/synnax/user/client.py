#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload
from uuid import UUID

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax.exceptions import ValidationError
from synnax.user.payload import New, User
from synnax.user.types_gen import RetrieveRequest, Where, WhereInput
from synnax.util import request_where, returns_one, sole
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


class _SetRolesRequest(BaseModel):
    keys: list[UUID]
    add: list[UUID]
    remove: list[UUID]


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
        roles: list[UUID] | None = None,
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
        roles: list[UUID] | None = None,
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
                roles=roles or [],
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

    def set_roles(
        self,
        keys: UUID | list[UUID],
        *,
        add: list[UUID] | None = None,
        remove: list[UUID] | None = None,
    ) -> None:
        """Removes roles from the users with the given keys, then adds roles to them.
        The Core applies both in one transaction, and a role in both lists is held
        after the call.

        :param keys: The keys of the users.
        :param add: The keys of the roles to add. Each one goes ahead of the roles a
            user keeps.
        :param remove: The keys of the roles to remove.
        """
        self.client.send(
            "/user/set-roles",
            _SetRolesRequest(keys=normalize(keys), add=add or [], remove=remove or []),
            Empty,
        )

    @overload
    def retrieve(
        self,
        *,
        key: UUID,
        usernames: str | list[str] = ...,
        roles: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> User: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[UUID] = ...,
        username: str,
        usernames: str | list[str] = ...,
        roles: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> User: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[UUID] = ...,
        usernames: str | list[str] = ...,
        roles: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[UUID] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> User: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[UUID] | filter.SetDict[UUID] | filter.Set[UUID] = ...,
        usernames: str | list[str] = ...,
        roles: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[UUID] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        offset: int = ...,
        one: Literal[False] = ...,
    ) -> list[User]: ...

    @require_named_params(example_params=("username", "synnax"))
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[UUID] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        one: bool = False,
        **fields: Any,
    ) -> User | list[User]:
        """Retrieves users from the Core. With no arguments, retrieves every user.

        Each filter field takes a value, a list of values, or an operator, and a user
        must match every field. A plain ``key`` or a ``username`` returns one user.

        :param key: Keeps users with this key.
        :param username: Keeps the user with this username.
        :param usernames: Keeps users with this username, or any username in the list.
        :param roles: Keeps the users that hold any of these roles.
        :param and_: Keeps users that match every filter in the list.
        :param or_: Keeps users that match any filter in the list.
        :param not_: Keeps users that do not match the filter.
        :param where: A filter tree that the users must also match.
        :param keys: Keeps users with one of these keys.
        :param search_term: A fuzzy search term matched against user names.
        :param limit: The maximum number of users to retrieve.
        :param offset: The number of users to skip.
        :param one: Returns one user for any filter.
        :returns: One user for a plain ``key``, a ``username``, or when ``one`` is
            set, otherwise a list of users. The list leaves out keys that do not
            exist.
        :raises NotFoundError: If one user is wanted and none matches.
        :raises MultipleFoundError: If one user is wanted and more than one matches.
        :raises ValidationError: If a filter field or ``username`` is None, or if one
            user is wanted beside a limit or an offset.
        """
        if "username" in fields:
            username = fields.pop("username")
            if username is None:
                raise ValidationError(
                    "username is None. Leave a field out to match every record."
                )
            where = filter.merge(where, Where(usernames=[username]))
            one = True
        single = returns_one(fields, one, limit, offset)
        flat = Where(key=None if keys is None else filter.Set[UUID](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            offset=offset,
        )
        res = self.client.send("/user/retrieve", req, _RetrieveResponse)
        return sole("user", req, res.users) if single else res.users

    def delete(self, keys: UUID | list[UUID] | None = None) -> None:
        self.client.send("/user/delete", _DeleteRequest(keys=normalize(keys)), Empty)
