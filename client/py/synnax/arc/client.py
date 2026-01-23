#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import overload
from uuid import UUID

from freighter import Empty, Payload, UnaryClient, send_required
from pydantic import PrivateAttr

from synnax.arc.payload import (
    ArcKey,
    ArcKeys,
    ArcMode,
    ArcName,
    ArcNames,
    ArcPayload,
    Graph,
    Text,
    ontology_id,
)
from synnax.exceptions import MultipleFoundError, NotFoundError
from synnax.ontology.payload import ID
from synnax.util.normalize import normalize


class _CreateRequest(Payload):
    arcs: list[ArcPayload] = []


class _CreateResponse(Payload):
    arcs: list[ArcPayload] = []


class _RetrieveRequest(Payload):
    keys: list[UUID] | None = None
    names: list[str] | None = None
    search_term: str | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveResponse(Payload):
    arcs: list[ArcPayload] | None = None


class _DeleteRequest(Payload):
    keys: list[UUID]


class Arc(ArcPayload):
    """An Arc is a user-defined automation sequence or control program.

    Arcs can be created in two modes:
    - 'text': Source code written in the Arc programming language
    - 'graph': Visual programming using a node-based editor
    """

    __client: ArcClient | None = PrivateAttr(None)

    def __init__(
        self,
        *,
        key: UUID = UUID(int=0),
        name: str = "",
        graph: Graph | None = None,
        text: Text | None = None,
        version: str = "",
        mode: ArcMode = "text",
        _client: ArcClient | None = None,
    ):
        super().__init__(
            key=key,
            name=name,
            graph=graph if graph is not None else Graph(),
            text=text if text is not None else Text(),
            version=version,
            mode=mode,
        )
        self.__client = _client

    @property
    def ontology_id(self) -> ID:
        return ontology_id(self.key)

    def to_payload(self) -> ArcPayload:
        return ArcPayload(
            key=self.key,
            name=self.name,
            graph=self.graph,
            text=self.text,
            version=self.version,
            mode=self.mode,
        )


class ArcClient:
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
        version: str = "",
        mode: ArcMode = "text",
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
        version: str = "",
        mode: ArcMode = "text",
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
                ArcPayload(
                    name=name,
                    graph=graph if graph is not None else Graph(),
                    text=text if text is not None else Text(),
                    version=version,
                    mode=mode,
                )
            ]

        res = send_required(
            self._client,
            "/arc/create",
            _CreateRequest(arcs=to_create),
            _CreateResponse,
        ).arcs
        created = self.__sugar(res)
        return created[0] if is_single else created

    @overload
    def retrieve(self, *, key: ArcKey) -> Arc: ...

    @overload
    def retrieve(self, *, name: ArcName) -> Arc: ...

    @overload
    def retrieve(
        self,
        *,
        keys: ArcKeys | None = None,
        names: ArcNames | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[Arc]: ...

    def retrieve(
        self,
        *,
        key: ArcKey | None = None,
        name: ArcName | None = None,
        keys: ArcKeys | None = None,
        names: ArcNames | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> Arc | list[Arc]:
        if key is not None:
            keys = [key]
        if name is not None:
            names = [name]

        is_single = key is not None or name is not None
        res = send_required(
            self._client,
            "/arc/retrieve",
            _RetrieveRequest(
                keys=keys,
                names=names,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            _RetrieveResponse,
        )

        arcs = self.__sugar(res.arcs or [])
        if not is_single:
            return arcs
        if len(arcs) == 0:
            identifier = key if key is not None else name
            raise NotFoundError(f"Arc {identifier} not found")
        if len(arcs) > 1:
            identifier = key if key is not None else name
            raise MultipleFoundError(f"Multiple Arcs matching {identifier} found")
        return arcs[0]

    def delete(self, keys: ArcKey | ArcKeys) -> None:
        send_required(
            self._client,
            "/arc/delete",
            _DeleteRequest(keys=normalize(keys)),
            Empty,
        )

    def __sugar(self, payloads: list[ArcPayload]) -> list[Arc]:
        return [
            Arc(
                key=p.key,
                name=p.name,
                graph=p.graph,
                text=p.text,
                version=p.version,
                mode=p.mode,
                _client=self,
            )
            for p in payloads
        ]
