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
from synnax.user.payload import NewUser, UserPayload
from synnax.util.normalize import normalize


class _CreateRequest(Payload):
    users: list[NewUser]


class _CreateResponse(Payload):
    users: list[UserPayload]


class _RetrieveRequest(Payload):
    keys: list[UUID] | None = None
    usernames: list[str] | None = None


class _RetrieveResponse(Payload):
    users: list[UserPayload] | None = None


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


class UserClient:
    client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        self.client = transport

    @overload
    def create(
        self, *, username: str, password: str, first_name: str = "", last_name: str = ""
    ) -> UserPayload: ...

    @overload
    def create(self, *, users: list[NewUser]) -> list[UserPayload]: ...

    def create(
        self,
        *,
        username: str | None,
        password: str | None,
        first_name: str | None,
        last_name: str | None,
        users: list[NewUser] | None,
    ) -> UserPayload | list[UserPayload]:
        if user is not None:
            users = normalize(user)
        res = send_required(
            self.client,
            _CREATE_ENDPOINT,
            _CreateRequest(users=users),
            _CreateResponse,
        ).users
        if user is not None:
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
        self, key: UUID, *, first_name: str | None = None, last_name: str | None = None
    ) -> None:
        if first_name is None:
            first_name = ""
        if last_name is None:
            last_name = ""
        send_required(
            self.client,
            _CHANGE_NAME_ENDPOINT,
            _ChangeNameRequest(key=key, first_name=first_name, last_name=last_name),
            Empty,
        )

    @overload
    def retrieve(self, *, key: UUID) -> UserPayload: ...

    @overload
    def retrieve(self, *, keys: list[UUID]) -> list[UserPayload]: ...

    @overload
    def retrieve(self, *, username: str) -> UserPayload: ...

    @overload
    def retrieve(self, *, usernames: list[str]) -> list[UserPayload]: ...

    def retrieve(
        self,
        *,
        key: UUID | None = None,
        keys: list[UUID] | None = None,
        username: str | None = None,
        usernames: list[str] | None = None,
    ) -> UserPayload | list[UserPayload]:
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

    @overload
    def delete(self, *, key: UUID) -> None: ...

    @overload
    def delete(self, *, keys: list[UUID]) -> None: ...

    def delete(
        self, *, key: UUID | None = None, keys: list[UUID] | None = None
    ) -> None:
        if key is not None:
            keys = normalize(key)
        send_required(
            self.client,
            _DELETE_ENDPOINT,
            _DeleteRequest(keys=keys),
            Empty,
        )
