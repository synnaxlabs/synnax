#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from freighter import Empty, Payload, UnaryClient, send_required

from synnax.group.payload import Group
from synnax.ontology.payload import ID, CrudeID


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
    keys: list[UUID]


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient):
        self._client = client

    def create(self, parent: CrudeID, name: str, key: str | None = None) -> Group:
        return send_required(
            self._client,
            "/ontology/create-group",
            CreateReq(parent=ID(parent), key=UUID(key) if key else None, name=name),
            CreateRes,
        ).group

    def rename(self, key: UUID, name: str) -> Empty:
        return send_required(
            self._client,
            "/ontology/rename-group",
            RenameReq(key=key, name=name),
            Empty,
        )

    def delete(self, keys: list[UUID]) -> Empty:
        return send_required(
            self._client,
            "/ontology/delete-group",
            DeleteReq(keys=keys),
            Empty,
        )
