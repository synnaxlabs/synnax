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
from synnax import ontology
from synnax.util.send_required import send_required


class Password(BaseModel):
    """A password as the API returns it: its subject and username, without its
    hash.
    """

    subject: ontology.ID
    """The user the password logs in as."""
    username: str
    """The name the password logs in with."""


class _CreateRequest(BaseModel):
    subject: ontology.ID | None = None
    username: str
    password: str


class _RetrieveRequest(BaseModel):
    subjects: list[ontology.ID] = []
    usernames: list[str] = []


class _RetrieveResponse(BaseModel):
    passwords: list[Password] = []


class _RenameRequest(BaseModel):
    subject: ontology.ID | None = None
    username: str


_CREATE_ENDPOINT = "/auth/password/create"
_RETRIEVE_ENDPOINT = "/auth/password/retrieve"
_RENAME_ENDPOINT = "/auth/password/rename"

RenameHandler = Callable[[ontology.ID | None, str], None]
"""Receives each rename the Core accepts, with the subject the caller gave and the new
username. A subject of None means the caller's own password."""


class Client:
    """Creates, lists, and renames passwords."""

    _client: UnaryClient
    _on_rename: RenameHandler

    def __init__(
        self,
        transport: UnaryClient,
        on_rename: RenameHandler = lambda _subject, _username: None,
    ) -> None:
        self._client = transport
        self._on_rename = on_rename

    def create(
        self, username: str, password: str, subject: ontology.ID | None = None
    ) -> None:
        """Gives the subject, or the caller when none is given, its first password.

        :raises AuthError: if another user holds the username.
        :raises AccessDenied: if the caller may not create the subject.
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
        subjects: list[ontology.ID] | None = None,
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

    def rename(self, username: str, subject: ontology.ID | None = None) -> None:
        """Changes the username of the subject's password, or of the caller's when
        none is given. Sessions stay open.

        :raises AuthError: if another user holds the username.
        :raises AccessDenied: if the caller may not update the subject.
        :raises NotFoundError: if the subject has no password.
        :raises ValidationError: if the username is empty.
        """
        send_required(
            self._client,
            _RENAME_ENDPOINT,
            _RenameRequest(subject=subject, username=username),
            Empty,
        )
        self._on_rename(subject, username)
