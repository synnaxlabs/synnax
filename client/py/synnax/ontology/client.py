#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload
from freighter import Payload, UnaryClient, send_required, Empty
from synnax.ontology.payload import ID, Resource, Relationship, CrudeID
from synnax.ontology.group import Client as GroupClient
from synnax.util.normalize import normalize
from pydantic import Field


class RetrieveReq(Payload):
    ids: list[ID]
    children: bool = False
    parents: bool = False
    include_schema: bool = False
    exclude_field_data: bool = False
    term: str | None = None
    limit: int | None = None
    offset: int | None = None
    types: list[str] | None = None


class AddChildrenReq(Payload):
    id: ID
    children: list[ID]


class RemoveChildrenReq(Payload):
    id: ID
    children: list[ID]


class MoveChildrenReq(Payload):
    from_: ID = Field(alias="from")
    to: ID
    children: list[ID]


_RETRIEVE_ENDPOINT = "/ontology/retrieve"
_ADD_CHILDREN_ENDPOINT = "/ontology/add-children"
_REMOVE_CHILDREN_ENDPOINT = "/ontology/remove-children"
_MOVE_CHILDREN_ENDPOINT = "/ontology/move-children"


class RetrieveRes(Payload):
    resources: list[Resource]


class Client:
    _client: UnaryClient
    groups: GroupClient

    def __init__(self, client: UnaryClient, groups: GroupClient) -> None:
        self._client = client
        self.groups = groups

    @overload
    def retrieve(
        self,
        id: CrudeID,
        *,
        children: bool = False,
        parents: bool = False,
        include_schema: bool = False,
        exclude_field_data: bool = False
    ) -> Resource:
        ...

    @overload
    def retrieve(
        self,
        ids: list[CrudeID],
        *,
        children: bool = False,
        parents: bool = False,
        include_schema: bool = False,
        exclude_field_data: bool = False
    ) -> list[Resource]:
        ...

    def retrieve(
        self,
        id: CrudeID | list[CrudeID],
        *,
        children: bool = False,
        parents: bool = False,
        include_schema: bool = False,
        exclude_field_data: bool = False
    ) -> Resource | list[Resource]:
        is_single = False
        if not isinstance(id, list):
            id = [id]
            is_single = True
        req = RetrieveReq(
            ids=[ID(i) for i in id],
            children=children,
            parents=parents,
            include_schema=include_schema,
            exclude_field_data=exclude_field_data,
        )
        res = send_required(self._client, _RETRIEVE_ENDPOINT, req, RetrieveRes)
        if is_single:
            return res.resources[0]
        return res.resources

    def retrieve_children(
        self,
        id: CrudeID | list[CrudeID],
    ) -> list[Resource]:
        return send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            RetrieveReq(ids=[ID(i) for i in normalize(id)], children=True),
            RetrieveRes,
        ).resources

    def retrieve_parents(
        self,
        id: CrudeID | list[CrudeID],
    ):
        return send_required(
            self._client,
            _RETRIEVE_ENDPOINT,
            RetrieveReq(ids=[ID(i) for i in normalize(id)], parents=True),
            RetrieveRes,
        ).resources

    def move_children(self, from_: CrudeID, to: CrudeID, *children: CrudeID) -> None:
        send_required(
            self._client,
            _MOVE_CHILDREN_ENDPOINT,
            MoveChildrenReq.parse_obj(
                {"from": ID(from_), "to": ID(to), "children": [ID(i) for i in children]}
            ),
            Empty,
        )

    def remove_children(self, id: CrudeID, *children: CrudeID) -> None:
        send_required(
            self._client,
            _REMOVE_CHILDREN_ENDPOINT,
            RemoveChildrenReq(id=ID(id), children=[ID(i) for i in children]),
            Empty,
        )

    def add_children(self, id: CrudeID, *children: CrudeID) -> None:
        send_required(
            self._client,
            _ADD_CHILDREN_ENDPOINT,
            AddChildrenReq(parent=ID(id), children=[ID(i) for i in children]),
            Empty,
        )
