#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax import resource
from synnax.auth.password.actions_gen import Action
from synnax.auth.password.types_gen import Key, Password
from synnax.util.send_required import send_required


class _CreateRequest(BaseModel):
    subject: resource.ID | None = None
    username: str
    password: str


class _RetrieveRequest(BaseModel):
    subjects: list[resource.ID] = []
    usernames: list[str] = []


class _RetrieveResponse(BaseModel):
    passwords: list[Password] = []


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[Action]


_CREATE_ENDPOINT = "/auth/password/create"
_RETRIEVE_ENDPOINT = "/auth/password/retrieve"
_DISPATCH_ENDPOINT = "/auth/password/dispatch"

UsernameHandler = Callable[[Key, str], None]
"""Receives each username change the Core accepts, with the key of the password."""


class Client:
    """Creates, lists, and changes passwords."""

    _client: UnaryClient
    _on_username: UsernameHandler

    def __init__(
        self,
        transport: UnaryClient,
        on_username: UsernameHandler = lambda _key, _username: None,
    ) -> None:
        self._client = transport
        self._on_username = on_username

    def create(
        self, username: str, password: str, subject: resource.ID | None = None
    ) -> None:
        """Gives the subject, or the caller when none is given, its first password.

        :raises AuthError: if another user holds the username.
        :raises AccessDenied: if the caller may not create and update the subject.
        :raises NotFoundError: if the subject does not exist.
        :raises ValidationError: if the username or password is empty, or the subject
            already has a password.
        """
        send_required(
            self._client,
            _CREATE_ENDPOINT,
            _CreateRequest(subject=subject, username=username, password=password),
            Empty,
        )

    def retrieve(
        self,
        subjects: list[resource.ID] | None = None,
        usernames: list[str] | None = None,
    ) -> list[Password]:
        """Lists the caller's password, or those of the given subjects or usernames
        when the caller may retrieve their subjects. Unknown subjects and usernames
        are ignored.

        :raises AccessDenied: if the caller may not retrieve another subject.
        """
        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(subjects=subjects or [], usernames=usernames or []),
            _RetrieveResponse,
        )
        return res.passwords

    def dispatch(self, key: Key, *actions: Action) -> None:
        """Applies the actions in order to the password with the given key. Sessions
        stay open. When the client logged in with the password, it logs in again under
        the new username.

        :raises AuthError: if another user holds the username that the actions set.
        :raises AccessDenied: if the caller may not update the password's user.
        :raises NotFoundError: if no password has the key.
        :raises ValidationError: if an action sets an empty username.
        """
        send_required(
            self._client,
            _DISPATCH_ENDPOINT,
            _DispatchRequest(key=key, actions=list(actions)),
            Empty,
        )
        for action in actions:
            self._on_username(key, action.set_username.username)
