#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax import ontology
from synnax.auth.apikey.types_gen import APIKey
from synnax.util.send_required import send_required


class _CreateRequest(BaseModel):
    subject: ontology.ID | None = None
    name: str


class _CreateResponse(BaseModel):
    token: str
    key: APIKey


class _RetrieveRequest(BaseModel):
    subjects: list[ontology.ID] = []


class _RetrieveResponse(BaseModel):
    keys: list[APIKey] = []


class _DeleteRequest(BaseModel):
    keys: list[UUID] = []


_CREATE_ENDPOINT = "/auth/api-key/create"
_RETRIEVE_ENDPOINT = "/auth/api-key/retrieve"
_DELETE_ENDPOINT = "/auth/api-key/delete"


class Client:
    """Creates, lists, and deletes API keys."""

    _client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        self._client = transport

    def create(
        self, name: str, subject: ontology.ID | None = None
    ) -> tuple[str, APIKey]:
        """Creates an API key for the subject, or for the caller when none is given.

        :returns: the token to log in with, the only copy of its secret, and the key.
        :raises AccessDenied: if the caller may not create a key for the subject.
        :raises NotFoundError: if the subject does not exist.
        """
        res = send_required(
            self._client,
            _CREATE_ENDPOINT,
            _CreateRequest(subject=subject, name=name),
            _CreateResponse,
        )
        return res.token, res.key

    def retrieve(self, subjects: list[ontology.ID] | None = None) -> list[APIKey]:
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

    def delete(self, *keys: UUID) -> None:
        """Deletes API keys and ends their sessions. Unknown keys are ignored."""
        send_required(
            self._client, _DELETE_ENDPOINT, _DeleteRequest(keys=list(keys)), Empty
        )
