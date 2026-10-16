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
from synnax.ontology.payload import ID, CrudeID, Resource
from synnax.ontology.types_gen import ResourceType
from x.lists import normalize


class RetrieveReq(BaseModel):
    ids: list[ID] = []
    search_term: str = ""
    types: list[ResourceType] = []
    limit: int = 0
    offset: int = 0
    children: bool = False
    parents: bool = False
    exclude_field_data: bool = False


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


class RetrieveRes(BaseModel):
    resources: list[Resource]


class Client:
    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    def retrieve(
        self,
        ids: CrudeID | list[CrudeID] | None = None,
        *,
        types: list[ResourceType] | None = None,
        limit: int = 0,
        offset: int = 0,
        exclude_field_data: bool = False,
    ) -> list[Resource]:
        """Retrieves resources from the ontology. Without ids or types, it retrieves
        all resources.

        :param ids: The IDs of the resources to retrieve.
        :param types: Only retrieve resources of these types.
        :param limit: The maximum number of resources to return. Zero means no limit.
        :param offset: The number of resources to skip.
        :param exclude_field_data: If true, the resources do not include their data.
        :returns: The matching resources.
        :raises NotFoundError: If a resource in ids does not exist.
        """
        return self._exec_retrieve(
            RetrieveReq(
                ids=_parse_ids(ids),
                types=types or [],
                limit=limit,
                offset=offset,
                exclude_field_data=exclude_field_data,
            )
        )

    def search(self, term: str) -> list[Resource]:
        """Fuzzy-searches the ontology for resources that match a term. The search
        index updates shortly after a resource changes.

        :param term: The term to search for.
        :returns: The matching resources.
        """
        return self._exec_retrieve(RetrieveReq(search_term=term))

    def retrieve_children(
        self,
        id: CrudeID | list[CrudeID],
        *,
        types: list[ResourceType] | None = None,
    ) -> list[ID]:
        """Retrieves the IDs of the children of one or more resources.

        :param id: The resources to retrieve the children of.
        :param types: Only retrieve children of these types.
        :returns: The IDs of the children.
        """
        req = RetrieveReq(
            ids=_parse_ids(id),
            types=types or [],
            children=True,
            exclude_field_data=True,
        )
        return [r.id for r in self._exec_retrieve(req)]

    def retrieve_parents(
        self,
        id: CrudeID | list[CrudeID],
        *,
        types: list[ResourceType] | None = None,
    ) -> list[ID]:
        """Retrieves the IDs of the parents of one or more resources.

        :param id: The resources to retrieve the parents of.
        :param types: Only retrieve parents of these types.
        :returns: The IDs of the parents.
        """
        req = RetrieveReq(
            ids=_parse_ids(id), types=types or [], parents=True, exclude_field_data=True
        )
        return [r.id for r in self._exec_retrieve(req)]

    def _exec_retrieve(self, req: RetrieveReq) -> list[Resource]:
        return self._client.send("/ontology/retrieve", req, RetrieveRes).resources

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


def _parse_ids(ids: CrudeID | list[CrudeID] | None) -> list[ID]:
    if ids is None:
        return []
    normalized: list[CrudeID] = normalize(ids)
    return [ID(i) for i in normalized]
