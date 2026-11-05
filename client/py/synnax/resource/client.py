#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel

from freighter import UnaryClient
from synnax.exceptions import NotFoundError
from synnax.resource.payload import ID, CrudeID
from synnax.resource.types_gen import Type
from x.lists import normalize


class Resource(BaseModel):
    """The ID and name of a record of any type."""

    id: ID
    name: str


class _SearchRequest(BaseModel):
    term: str
    types: list[Type] | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveRequest(BaseModel):
    ids: list[ID]


class _Response(BaseModel):
    resources: list[Resource]


class Client:
    """Searches and names records of every type on a Core."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    def search(
        self,
        term: str,
        *,
        types: list[Type] | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[Resource]:
        """Searches the records of every type for a term.

        :param term: The text that a resource must match.
        :param types: Limits the search to resources of these types.
        :param limit: The maximum number of resources to return. Defaults to 100.
        :param offset: The number of closest matches to skip.
        :returns: The matching resources, closest match first.
        :raises ValidationError: If a type is unknown, or the limit or offset is
            negative.
        """
        req = _SearchRequest(term=term, types=types, limit=limit, offset=offset)
        return self._client.send("/resource/search", req, _Response).resources

    @overload
    def retrieve(self, ids: CrudeID) -> Resource: ...

    @overload
    def retrieve(self, ids: list[CrudeID]) -> list[Resource]: ...

    def retrieve(self, ids: CrudeID | list[CrudeID]) -> Resource | list[Resource]:
        """Retrieves the ID and name of each resource.

        :param ids: A single resource ID or a list of them.
        :returns: The resource, or for a list, the resources in the order of the IDs.
            The list leaves out each ID that names no resource.
        :raises NotFoundError: If a single ID names no resource.
        :raises ValidationError: If an ID has an unknown type.
        """
        is_single = not isinstance(ids, list)
        normalized: list[CrudeID] = normalize(ids)
        req = _RetrieveRequest(ids=[ID(i) for i in normalized])
        res = self._client.send("/resource/retrieve", req, _Response)
        if not is_single:
            return res.resources
        if len(res.resources) == 0:
            raise NotFoundError(f"Resource {ids} not found")
        return res.resources[0]
