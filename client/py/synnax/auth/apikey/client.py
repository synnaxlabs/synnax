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
from synnax.auth.apikey.types_gen import APIKey, Key
from synnax.util.send_required import send_required
from x.telem import CrudeTimeStamp, TimeSpan, TimeStamp


class _CreateRequest(BaseModel):
    subject: resource.ID | None = None
    name: str
    expires: TimeStamp


class _CreateResponse(BaseModel):
    token: str
    key: APIKey


class _RetrieveRequest(BaseModel):
    subjects: list[resource.ID] = []


class _RetrieveResponse(BaseModel):
    keys: list[APIKey] = []


class _DeleteRequest(BaseModel):
    keys: list[Key] = []


# A client clock that runs ahead of the Core's must not push a key past the limit.
_EXPIRY_MARGIN = TimeSpan.HOUR


def longest_expiry(max_lifetime: TimeSpan | None) -> TimeStamp:
    """Returns the longest expiry to ask for: TimeStamp.MAX when max_lifetime is None,
    else an hour short of now plus max_lifetime, or half of it short when it is under
    two hours.
    """
    if max_lifetime is None:
        return TimeStamp.MAX
    margin = (
        TimeSpan(max_lifetime // 2)
        if max_lifetime < _EXPIRY_MARGIN * 2
        else _EXPIRY_MARGIN
    )
    return TimeStamp.now() + max_lifetime - margin


_CREATE_ENDPOINT = "/auth/api-key/create"
_RETRIEVE_ENDPOINT = "/auth/api-key/retrieve"
_DELETE_ENDPOINT = "/auth/api-key/delete"
_ROTATE_ENDPOINT = "/auth/api-key/rotate"


class Client:
    """Creates, lists, rotates, and deletes API keys."""

    _client: UnaryClient
    _on_rotate: Callable[[str, APIKey], None]

    def __init__(
        self,
        transport: UnaryClient,
        on_rotate: Callable[[str, APIKey], None] = lambda _token, _key: None,
    ) -> None:
        """
        :param on_rotate: receives the new token and the key after each rotation.
        """
        self._client = transport
        self._on_rotate = on_rotate

    def create(
        self,
        name: str,
        subject: resource.ID | None = None,
        expires: CrudeTimeStamp = TimeStamp.MAX,
    ) -> tuple[str, APIKey]:
        """Creates an API key for the subject, or for the caller when none is given.

        :param expires: when the key stops logging in. Defaults to TimeStamp.MAX,
            which never expires.
        :returns: the token to log in with, the only copy of its secret, and the key.
        :raises AccessDenied: if the caller may not create a key for the subject.
        :raises NotFoundError: if the subject does not exist.
        :raises ValidationError: if expires is not in the future.
        """
        res = send_required(
            self._client,
            _CREATE_ENDPOINT,
            _CreateRequest(
                subject=subject,
                name=name,
                expires=TimeStamp(expires),
            ),
            _CreateResponse,
        )
        return res.token, res.key

    def rotate(self) -> tuple[str, APIKey]:
        """Gives the API key that the client logged in with a new secret. The key lasts
        as long as it did before, or an hour short of the lifetime cap of the Core. The
        client's session stays open and later logins use the new token. The old token
        cannot log in again.

        :returns: the new token, the only copy of the new secret, and the key.
        :raises ValidationError: if the client did not log in with an API key.
        :raises AuthError: if the key has expired.
        """
        res = send_required(self._client, _ROTATE_ENDPOINT, Empty(), _CreateResponse)
        self._on_rotate(res.token, res.key)
        return res.token, res.key

    def retrieve(self, subjects: list[resource.ID] | None = None) -> list[APIKey]:
        """Lists the caller's API keys, or those of the given subjects when the caller
        may retrieve them.
        """
        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(subjects=subjects or []),
            _RetrieveResponse,
        )
        return res.keys

    def delete(self, *keys: Key) -> None:
        """Deletes API keys and ends their sessions. Unknown keys are ignored."""
        send_required(
            self._client, _DELETE_ENDPOINT, _DeleteRequest(keys=list(keys)), Empty
        )
