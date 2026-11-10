#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax.access import role as role_
from synnax.exceptions import ValidationError
from synnax.transport import send_batch
from synnax.user import actions_gen
from synnax.user.payload import New, User
from synnax.user.types_gen import Cursor, Key, Order, RetrieveRequest, Where, WhereInput
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
    keys: list[Key]


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


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
        role: role_.Key,
        name: str = "",
        key: Key | None = None,
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
        role: role_.Key | None = None,
        name: str = "",
        key: Key | None = None,
        user: New | None = None,
        users: list[New] | None = None,
    ) -> User | list[User]:
        if username is not None:
            if password is None:
                raise ValueError("password is required when username is provided")
            if role is None:
                raise ValueError("role is required when username is provided")
            user = New(
                username=username, password=password, name=name, key=key, role=role
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

    def change_username(self, key: Key, username: str) -> None:
        action = actions_gen.set_username(
            actions_gen.SetUsernamePayload(username=username)
        )
        self.client.send(
            "/user/dispatch", _DispatchRequest(key=key, actions=[action]), Empty
        )

    def change_name(self, key: Key, *, name: str) -> None:
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        self.client.send(
            "/user/dispatch", _DispatchRequest(key=key, actions=[action]), Empty
        )

    def set_role(self, keys: Key | list[Key], role: role_.Key) -> None:
        """Gives a role to the users with the given keys. The role replaces the role
        each user holds. The Core changes all of the users in one transaction.

        :param keys: The keys of the users.
        :param role: The key of the role to give.
        :raises ValidationError: If one of the users is a root user or the role does
            not exist.
        :raises NotFoundError: If one of the users does not exist.
        """
        action = actions_gen.set(actions_gen.SetPayload(path=["role"], value=str(role)))
        send_batch(
            self.client,
            "/user/dispatch",
            [_DispatchRequest(key=k, actions=[action]) for k in normalize(keys)],
        )

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        usernames: str | list[str] = ...,
        roles: role_.Key | list[role_.Key] = ...,
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
        key: filter.SetInput[Key] = ...,
        username: str,
        usernames: str | list[str] = ...,
        roles: role_.Key | list[role_.Key] = ...,
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
        key: filter.SetInput[Key] = ...,
        usernames: str | list[str] = ...,
        roles: role_.Key | list[role_.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> User: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        usernames: str | list[str] = ...,
        roles: role_.Key | list[role_.Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[User]: ...

    @require_named_params(example_params=("username", "synnax"))
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
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
        :param order: The order of the users. Defaults to key order.
        :param after: Retrieves the users past this position in the order: the key of
            the last user of the page before, and its value of the order field.
        :param one: Returns one user for any filter.
        :returns: One user for a plain ``key``, a ``username``, or when ``one`` is
            set, otherwise a list of users. The list leaves out keys that do not
            exist.
        :raises NotFoundError: If one user is wanted and none matches.
        :raises MultipleFoundError: If one user is wanted and more than one matches.
        :raises ValidationError: If a filter field or ``username`` is None, or if one
            user is wanted beside a limit or a cursor.
        """
        if "username" in fields:
            username = fields.pop("username")
            if username is None:
                raise ValidationError(
                    "username is None. Leave a field out to match every record."
                )
            where = filter.merge(where, Where(usernames=[username]))
            one = True
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self.client.send("/user/retrieve", req, _RetrieveResponse)
        return sole("user", req, res.users) if single else res.users

    def delete(self, keys: Key | list[Key] | None = None) -> None:
        self.client.send("/user/delete", _DeleteRequest(keys=normalize(keys)), Empty)
