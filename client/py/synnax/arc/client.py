#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Any, Literal, overload
from uuid import UUID

from pydantic import BaseModel, PrivateAttr

from freighter import Empty, UnaryClient
from synnax.arc.graph import Graph
from synnax.arc.program import Program
from synnax.arc.text import Text
from synnax.arc.types_gen import (
    Key,
    Mode,
    Payload,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
    ontology_id,
)
from synnax.ontology.payload import ID
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize
from x.uuid import NIL


class _CreateRequest(BaseModel):
    arcs: list[Payload] = []


class _CreateResponse(BaseModel):
    arcs: list[Payload] = []


class _DeleteRequest(BaseModel):
    keys: list[UUID]


class Arc(Payload):
    """An Arc is a user-defined automation sequence or control program.

    Arcs can be created in two modes:
    - 'text': Source code written in the Arc programming language
    - 'graph': Visual programming using a node-based editor
    """

    _client: Client | None = PrivateAttr(None)

    def __init__(
        self,
        *,
        key: UUID = NIL,
        name: str = "",
        graph: Graph | None = None,
        text: Text | None = None,
        mode: Mode = "text",
        program: Program | None = None,
        _client: Client | None = None,
    ):
        super().__init__(
            key=key,
            name=name,
            graph=graph if graph is not None else Graph(),
            text=text if text is not None else Text(),
            mode=mode,
            program=program,
        )
        self._client = _client

    @property
    def ontology_id(self) -> ID:
        return ontology_id(self.key)

    def to_payload(self) -> Payload:
        return Payload(
            key=self.key,
            name=self.name,
            graph=self.graph,
            text=self.text,
            mode=self.mode,
        )


class Client:
    """Client for managing Arc programs in the Synnax cluster."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(
        self,
        *,
        name: str,
        graph: Graph | None = None,
        text: Text | None = None,
        mode: Mode = "text",
    ) -> Arc: ...

    @overload
    def create(self, arc: Arc) -> Arc: ...

    @overload
    def create(self, arc: list[Arc]) -> list[Arc]: ...

    def create(
        self,
        arc: Arc | list[Arc] | None = None,
        *,
        name: str = "",
        graph: Graph | None = None,
        text: Text | None = None,
        mode: Mode = "text",
    ) -> Arc | list[Arc]:
        is_single = True
        if arc is not None:
            if isinstance(arc, list):
                is_single = False
                to_create = [a.to_payload() for a in arc]
            else:
                to_create = [arc.to_payload()]
        else:
            to_create = [
                Payload(
                    name=name,
                    graph=graph if graph is not None else Graph(),
                    text=text if text is not None else Text(),
                    mode=mode,
                )
            ]

        res = self._client.send(
            "/arc/create", _CreateRequest(arcs=to_create), _CreateResponse
        ).arcs
        created = self._sugar(res)
        return created[0] if is_single else created

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_program: bool = ...,
        one: bool = ...,
    ) -> Arc: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_program: bool = ...,
        one: bool = ...,
    ) -> Arc: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        include_program: bool = ...,
        one: Literal[True],
    ) -> Arc: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        offset: int = ...,
        include_program: bool = ...,
        one: Literal[False] = ...,
    ) -> list[Arc]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        include_program: bool = False,
        one: bool = False,
        **fields: Any,
    ) -> Arc | list[Arc]:
        """Retrieves Arcs from the Core. With no arguments, retrieves every Arc.

        Each filter field takes a value, a list of values, or an operator, and an Arc
        must match every field. A plain ``key`` or ``name`` returns one Arc.

        :param key: Keeps Arcs with this key.
        :param name: Keeps Arcs with this name.
        :param and_: Keeps Arcs that match every filter in the list.
        :param or_: Keeps Arcs that match any filter in the list.
        :param not_: Keeps Arcs that do not match the filter.
        :param where: A filter tree that the Arcs must also match.
        :param keys: Keeps Arcs with one of these keys.
        :param names: Keeps Arcs with one of these names.
        :param search_term: A fuzzy search term matched against Arc names.
        :param limit: The maximum number of Arcs to retrieve.
        :param offset: The number of Arcs to skip.
        :param include_program: Whether the Core compiles each Arc and sets its
            program. The Core raises an error if any Arc fails to compile.
        :param one: Returns one Arc for any filter.
        :returns: One Arc for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of Arcs. The list leaves out keys that do not exist.
        :raises NotFoundError: If one Arc is wanted and none matches.
        :raises MultipleFoundError: If one Arc is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one Arc is wanted
            beside a limit or an offset.
        """
        single = returns_one(fields, one, limit, offset)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
        )
        res = self._client.send(
            "/arc/retrieve",
            RetrieveRequest(
                where=request_where(Where, filter.merge(where, flat), fields),
                search_term=search_term,
                limit=limit,
                offset=offset,
                include_program=include_program,
            ),
            RetrieveResponse,
        )
        arcs = self._sugar(res.arcs)
        return sole("Arc", fields, arcs) if single else arcs

    def delete(self, keys: Key | list[Key]) -> None:
        self._client.send("/arc/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def _sugar(self, payloads: list[Payload]) -> list[Arc]:
        return [
            Arc(
                key=p.key,
                name=p.name,
                graph=p.graph,
                text=p.text,
                mode=p.mode,
                program=p.program,
                _client=self,
            )
            for p in payloads
        ]
