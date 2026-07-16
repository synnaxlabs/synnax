#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax.ontology.payload import ID, CrudeID
from x.lists import normalize


class RetrieveReq(BaseModel):
    ids: list[ID]
    children: bool = False
    parents: bool = False
    exclude_field_data: bool = True


class AddChildrenReq(BaseModel):
    id: ID
    children: list[ID]


class RemoveChildrenReq(BaseModel):
    id: ID
    children: list[ID]


class MoveChildrenReq(BaseModel):
    from_: ID = Field(alias="from")
    to: ID
    children: list[ID]


class _Resource(BaseModel):
    id: ID


class RetrieveRes(BaseModel):
    resources: list[_Resource]


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    def retrieve_children(
        self,
        id: CrudeID | list[CrudeID],
    ) -> list[ID]:
        normalized: list[CrudeID] = normalize(id)
        return self._exec_retrieve(
            RetrieveReq(ids=[ID(i) for i in normalized], children=True)
        )

    def _exec_retrieve(self, req: RetrieveReq) -> list[ID]:
        res = self._client.send("/ontology/retrieve", req, RetrieveRes)
        return [r.id for r in res.resources]

    def retrieve_parents(
        self,
        id: CrudeID | list[CrudeID],
    ) -> list[ID]:
        normalized: list[CrudeID] = normalize(id)
        return self._exec_retrieve(
            RetrieveReq(ids=[ID(i) for i in normalized], parents=True)
        )

    def move_children(self, from_: CrudeID, to: CrudeID, *children: CrudeID) -> None:

        self._client.send(
            "/ontology/move-children",
            MoveChildrenReq.model_validate(
                {"from": ID(from_), "to": ID(to), "children": [ID(i) for i in children]}
            ),
            Empty,
        )

    def remove_children(self, id: CrudeID, *children: CrudeID) -> None:
        self._client.send(
            "/ontology/remove-children",
            RemoveChildrenReq(id=ID(id), children=[ID(i) for i in children]),
            Empty,
        )

    def add_children(self, id: CrudeID, *children: CrudeID) -> None:
        self._client.send(
            "/ontology/add-children",
            AddChildrenReq(id=ID(id), children=[ID(i) for i in children]),
            Empty,
        )
