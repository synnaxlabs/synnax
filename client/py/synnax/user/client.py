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

from freighter import Empty, UnaryClient, send_required
from pydantic import BaseModel

from synnax.exceptions import NotFoundError
from synnax.user.payload import New, User
from synnax.util.normalize import normalize
from synnax.util.params import require_named_params


class _CreateRequest(BaseModel):
    users: list[New]


class _CreateResponse(BaseModel):
    users: list[User]


class _RetrieveRequest(BaseModel):
    keys: list[UUID] | None = None
    usernames: list[str] | None = None


class _RetrieveResponse(BaseModel):
    users: list[User] | None = None


class _DeleteRequest(BaseModel):
    keys: list[UUID]


class _ChangeUsernameRequest(BaseModel):
    key: UUID
    username: str


class _ChangeNameRequest(BaseModel):
    key: UUID
    first_name: str
    last_name: str


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
        first_name: str = "",
        last_name: str = "",
        key: UUID | None = None,
    ) -> User: ...

    @overload
    def create(self, *, user: New) -> User: ...

    @overload
    def create(self, *, users: list[New]) -> list[User]: ...

    @require_named_params(example_params=("user", "NewUser(username='synnax')"))
    def create(
        self,
        *,
        username: str | None = None,
        password: str | None = None,
        first_name: str | None = None,
        last_name: str | None = None,
        key: UUID | None = None,
        user: New | None = None,
        users: list[New] | None = None,
    ) -> User | list[User]:
        if username is not None:
            if first_name is None:
                first_name = ""
            if last_name is None:
                last_name = ""
            user = New(
                username=username,
                password=password,
                first_name=first_name,
                last_name=last_name,
                key=key,
            )
        single = user is not None
        if user is not None:
            users = [user]
        if users is None:
            raise ValueError("Either username, user, or users must be provided")
        res = send_required(
            self.client,
            "/user/create",
            _CreateRequest(users=users),
            _CreateResponse,
        ).users
        if single:
            return res[0]
        return res

    def change_username(self, key: UUID, username: str) -> None:
        send_required(
            self.client,
            "/user/change_username",
            _ChangeUsernameRequest(key=key, username=username),
            Empty,
        )

    def change_name(
        self, key: UUID, *, first_name: str = "", last_name: str = ""
    ) -> None:
        send_required(
            self.client,
            "/user/change_name",
            _ChangeNameRequest(key=key, first_name=first_name, last_name=last_name),
            Empty,
        )

    @overload
    def retrieve(self, *, key: UUID) -> User: ...

    @overload
    def retrieve(self, *, keys: list[UUID]) -> list[User]: ...

    @overload
    def retrieve(self, *, username: str) -> User: ...

    @overload
    def retrieve(self, *, usernames: list[str]) -> list[User]: ...

    @require_named_params(example_params=("username", "synnax"))
    def retrieve(
        self,
        *,
        key: UUID | None = None,
        keys: list[UUID] | None = None,
        username: str | None = None,
        usernames: list[str] | None = None,
    ) -> User | list[User]:
        if key is not None:
            keys = normalize(key)
        if username is not None:
            usernames = normalize(username)
        single = key is not None or username is not None
        res = send_required(
            self.client,
            "/user/retrieve",
            _RetrieveRequest(keys=keys, usernames=usernames),
            _RetrieveResponse,
        )
        users = res.users or []
        if not single:
            return users
        if len(users) == 0:
            raise NotFoundError(f"User matching {key or username} not found")
        return users[0]

    def delete(self, keys: UUID | list[UUID] | None = None) -> None:
        send_required(
            self.client,
            "/user/delete",
            _DeleteRequest(keys=normalize(keys)),
            Empty,
        )
