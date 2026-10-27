#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Literal, overload
from uuid import UUID

from pydantic import BaseModel, PrivateAttr

from freighter import Empty, UnaryClient
from synnax import util
from synnax.arc import builder_gen
from synnax.arc.builder_gen import Condition
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
    ontology_id,
)
from synnax.ontology.payload import ID
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
    def retrieve(self, key: Key | str, /, *, include_program: bool = False) -> Arc: ...

    @overload
    def retrieve(
        self, keys: list[Key] | list[str], /, *, include_program: bool = False
    ) -> list[Arc]: ...

    @overload
    def retrieve(self, *, key: Key, include_program: bool = False) -> Arc: ...

    @overload
    def retrieve(self, *, name: str, include_program: bool = False) -> Arc: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[True],
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        include_program: bool = False,
    ) -> Arc: ...

    @overload
    def retrieve(
        self,
        *conditions: Condition,
        one: Literal[False] = False,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        include_program: bool = False,
    ) -> list[Arc]: ...

    def retrieve(
        self,
        *args: Key | str | list[Key] | list[str] | Condition,
        key: Key | None = None,
        name: str | None = None,
        one: bool = False,
        keys: list[Key] | None = None,
        names: list[str] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
        include_program: bool = False,
    ) -> Arc | list[Arc]:
        """Retrieves Arcs that match every condition and filter. Returns one Arc for a
        key, a name, or one, and a list otherwise.

        :param args: Conditions from the Arc builder, such as
            ``arc.name.contains("press")``, or one key, name, or list of either.
        :param one: Returns the one Arc that matches, instead of a list.
        :param include_program: Whether the Core compiles each Arc and sets its
            program. The Core raises an error if any Arc fails to compile.
        :raises NotFoundError: If a single Arc is requested and none matches.
        :raises MultipleFoundError: If a single Arc is requested by name or one and more
            than one matches.
        :raises TypeError: If args are neither Arc conditions nor a shortcut.
        """
        conditions, query = util.shortcut(
            "Arc", args, Condition, UUID, builder_gen.key, builder_gen.name
        )
        if key is not None:
            keys = [key]
        if name is not None:
            names = [name]
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
        )
        if key is not None or name is not None:
            query = key if key is not None else name
        where = filter.merge(filter.join(Where, conditions), flat)
        res = self._client.send(
            "/arc/retrieve",
            RetrieveRequest(
                where=where,
                search_term=search_term,
                limit=limit,
                offset=offset,
                include_program=include_program,
            ),
            RetrieveResponse,
        )

        arcs = self._sugar(res.arcs)
        if query is None and not one:
            return arcs
        return util.one("Arc", where, arcs)

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
