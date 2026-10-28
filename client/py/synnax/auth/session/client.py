#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax import ontology
from synnax.auth.types_gen import Session, SessionKey
from synnax.util.send_required import send_required


class _RetrieveRequest(BaseModel):
    subjects: list[ontology.ID] = []


class _RetrieveResponse(BaseModel):
    sessions: list[Session] = []


class _DeleteRequest(BaseModel):
    keys: list[SessionKey] = []


_RETRIEVE_ENDPOINT = "/auth/session/retrieve"
DELETE_ENDPOINT = "/auth/session/delete"


class Client:
    """Lists and ends sessions."""

    _client: UnaryClient

    def __init__(self, transport: UnaryClient) -> None:
        self._client = transport

    def retrieve(self, subjects: list[ontology.ID] | None = None) -> list[Session]:
        """Lists the caller's sessions, or those of the given subjects when the caller
        may retrieve them.
        """
        res = send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            _RetrieveRequest(subjects=subjects or []),
            _RetrieveResponse,
        )
        return res.sessions

    def delete(self, *keys: SessionKey) -> None:
        """Ends the given sessions. Unknown keys are ignored."""
        send_required(
            self._client, DELETE_ENDPOINT, _DeleteRequest(keys=list(keys)), Empty
        )
