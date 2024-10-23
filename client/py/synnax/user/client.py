#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID
from freighter import UnaryClient, send_required, Payload, Empty
from typing import overload
from synnax.user.payload import NewUser, User
from synnax.util.normalize import normalize


class _CreateRequest(Payload):
    users: list[NewUser]


class _CreateResponse(Payload):
    users: list[User]


class _RetrieveRequest(Payload):
    keys: list[UUID] | None = None
    usernames: list[str] | None = None


class _RetrieveResponse(Payload):
    users: list[User] | None = None


class _DeleteRequest(Payload):
    keys: list[UUID]


class _ChangeUsernameRequest(Payload):
    key: UUID
    username: str


class _ChangeNameRequest(Payload):
    key: UUID
    first_name: str
    last_name: str


_CREATE_ENDPOINT = "/user/create"
_CHANGE_USERNAME_ENDPOINT = "/user/change_username"
_CHANGE_NAME_ENDPOINT = "/user/change_name"
_DELETE_ENDPOINT = "/user/delete"
_RETRIEVE_ENDPOINT = "/user/retrieve"


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
    ) -> User:
        ...

    @overload
    def create(self, *, user: NewUser) -> User:
        ...

    @overload
    def create(self, *, users: list[NewUser]) -> list[User]:
        ...

    def create(
        self,
        *,
        username: str | None = None,
        password: str | None = None,
        first_name: str | None = None,
        last_name: str | None = None,
        key: UUID | None = None,
        user: NewUser | None = None,
        users: list[NewUser] | None = None,
    ) -> User | list[User]:
        if username is not None:
            if first_name is None:
                first_name = ""
            if last_name is None:
                last_name = ""
            user = NewUser(
                username=username,
                password=password,
                first_name=first_name,
                last_name=last_name,
                key=key,
            )
        single = user is not None
        if single:
            users = [user]
        res = send_required(
            self.client,
            _CREATE_ENDPOINT,
            _CreateRequest(users=users),
            _CreateResponse,
        ).users
        if single:
            return res[0]
        return res

    def change_username(self, key: UUID, username: str) -> None:
        send_required(
            self.client,
            _CHANGE_USERNAME_ENDPOINT,
            _ChangeUsernameRequest(key=key, username=username),
            Empty,
        )

    def change_name(
        self, key: UUID, *, first_name: str = "", last_name: str = ""
    ) -> None:
        send_required(
            self.client,
            _CHANGE_NAME_ENDPOINT,
            _ChangeNameRequest(key=key, first_name=first_name, last_name=last_name),
            Empty,
        )

    @overload
    def retrieve(self, *, key: UUID) -> User:
        ...

    @overload
    def retrieve(self, *, keys: list[UUID]) -> list[User]:
        ...

    @overload
    def retrieve(self, *, username: str) -> User:
        ...

    @overload
    def retrieve(self, *, usernames: list[str]) -> list[User]:
        ...

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
        return send_required(
            self.client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(keys=keys, usernames=usernames),
            _RetrieveResponse,
        ).users

    def delete(self, keys: UUID | list[UUID] | None = None) -> None:
        send_required(
            self.client,
            _DELETE_ENDPOINT,
            _DeleteRequest(keys=normalize(keys)),
            Empty,
        )
