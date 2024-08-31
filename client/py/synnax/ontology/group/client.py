#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID
from freighter import UnaryClient, send_required, Empty, Payload
from synnax.ontology.payload import ID, CrudeID
from synnax.ontology.group.payload import Group


class CreateReq(Payload):
    parent: ID
    key: UUID | None = None
    name: str


class CreateRes(Payload):
    group: Group


class RenameReq(Payload):
    key: UUID
    name: str


class DeleteReq(Payload):
    key: list[UUID]


_CREATE_ENDPOINT = "/ontology/create-group"
_RENAME_ENDPOINT = "/ontology/rename-group"
_DELETE_ENDPOINT = "/ontology/delete-group"


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient):
        self._client = client

    def create(self, parent: CrudeID, name: str, key: str | None = None) -> Group:
        return send_required(
            self._client,
            _CREATE_ENDPOINT,
            CreateReq(parent=ID(parent), key=UUID(key) if key else None, name=name),
            CreateRes,
        ).group

    def rename(self, key: CrudeID, name: str) -> Empty:
        return send_required(
            self._client, _RENAME_ENDPOINT, RenameReq(key=ID(key), name=name), Empty
        )

    def delete(self, keys: list[CrudeID]) -> Empty:
        return send_required(
            self._client,
            _DELETE_ENDPOINT,
            DeleteReq(key=[ID(key) for key in keys]),
            Empty,
        )
