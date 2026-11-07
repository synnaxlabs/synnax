#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Sequence
from typing import Any, Literal, overload
from uuid import UUID

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.arc.graph import Graph
from synnax.arc.text import Text
from synnax.arc.types_gen import (
    AutomationPayload,
    GraphSource,
    Key,
    Mode,
    ModulePayload,
    Payload,
    RetrieveRequest,
    RetrieveResponse,
    Source,
    TextSource,
    Type,
    Where,
    WhereInput,
    ontology_id,
)
from synnax.ontology.payload import ID
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    arcs: list[Payload] = []


class _CreateResponse(BaseModel):
    arcs: list[Payload] = []


class _DeleteRequest(BaseModel):
    keys: list[UUID]


class Automation(AutomationPayload):
    """An Arc that runs on a rack as a task. It is written in one of two modes:

    - 'text': Source code written in the Arc programming language
    - 'graph': Visual programming using a node-based editor
    """

    @property
    def ontology_id(self) -> ID:
        return ontology_id(self.key)


class Module(ModulePayload):
    """An Arc that holds code for other Arcs to import."""

    @property
    def ontology_id(self) -> ID:
        return ontology_id(self.key)


Arc = Automation | Module

_PAYLOADS: dict[Type, type[AutomationPayload] | type[ModulePayload]] = {
    "automation": AutomationPayload,
    "module": ModulePayload,
}
_ARCS: dict[Type, type[Automation] | type[Module]] = {
    "automation": Automation,
    "module": Module,
}


def _source(mode: Mode, text: Text | None, graph: Graph | None) -> Source:
    if mode == "graph":
        if text is not None:
            raise ValueError("a graph-mode Arc takes no text")
        return GraphSource(graph=graph if graph is not None else Graph())
    if graph is not None:
        raise ValueError("a text-mode Arc takes no graph")
    return TextSource(text=text if text is not None else Text())


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
        type: Type = "automation",
    ) -> Arc: ...

    @overload
    def create(self, arc: Arc) -> Arc: ...

    @overload
    def create(self, arc: Sequence[Arc]) -> list[Arc]: ...

    def create(
        self,
        arc: Arc | Sequence[Arc] | None = None,
        *,
        name: str = "",
        graph: Graph | None = None,
        text: Text | None = None,
        mode: Mode = "text",
        type: Type = "automation",
    ) -> Arc | list[Arc]:
        is_single = not isinstance(arc, Sequence)
        to_create: list[Payload]
        if isinstance(arc, Sequence):
            to_create = list(arc)
        elif arc is not None:
            to_create = [arc]
        else:
            to_create = [_PAYLOADS[type](name=name, source=_source(mode, text, graph))]

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
        :param include_program: Whether the Core compiles each automation and sets
            its program. The Core raises an error if any automation fails to compile.
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
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            offset=offset,
            include_program=include_program,
        )
        res = self._client.send("/arc/retrieve", req, RetrieveResponse)
        arcs = self._sugar(res.arcs)
        return sole("Arc", req.where, arcs) if single else arcs

    def delete(self, keys: Key | list[Key]) -> None:
        self._client.send("/arc/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def _sugar(self, payloads: list[Payload]) -> list[Arc]:
        return [_ARCS[p.type].model_validate(p, from_attributes=True) for p in payloads]
