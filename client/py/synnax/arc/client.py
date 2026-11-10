#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import random
from collections.abc import Sequence
from typing import Any, Literal, overload
from uuid import UUID

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax import rack as rack_, task as task_
from synnax.arc import actions_gen, directory
from synnax.arc.graph import Graph
from synnax.arc.task import Performance
from synnax.arc.text import Text
from synnax.arc.types_gen import (
    AutomationPayload,
    Cursor,
    GraphSource,
    Key,
    Mode,
    ModulePayload,
    Order,
    Payload,
    RetrieveRequest,
    RetrieveResponse,
    Source,
    TextSource,
    Type,
    Where,
    WhereInput,
    resource_id,
)
from synnax.resource import ID
from synnax.util import request_where, returns_one, sole
from x import crdt, filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    arcs: list[Payload] = []


class _CreateResponse(BaseModel):
    arcs: list[Payload] = []


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _UpdateTaskRequest(BaseModel):
    key: Key
    rack: rack_.Key | None = None
    performance: Performance | None = None


class _UpdateTaskResponse(BaseModel):
    task: task_.Payload | None = None


class Automation(AutomationPayload):
    """An Arc that runs on a rack as a task. It is written in one of two modes:

    - 'text': Source code written in the Arc programming language
    - 'graph': Visual programming using a node-based editor
    """

    @property
    def resource_id(self) -> ID:
        return resource_id(self.key)


class Module(ModulePayload):
    """An Arc that holds code for other Arcs to import."""

    @property
    def resource_id(self) -> ID:
        return resource_id(self.key)


Arc = Automation | Module

_PAYLOADS: dict[Type, type[AutomationPayload] | type[ModulePayload]] = {
    "automation": AutomationPayload,
    "module": ModulePayload,
}
_ARCS: dict[Type, type[Automation] | type[Module]] = {
    "automation": Automation,
    "module": Module,
}


# The Core writes the text of a created Arc as this replica, so an edit takes another.
_SEED_REPLICA = 1
_MAX_REPLICA = 0xFFFFFFFF
_ROOT = crdt.ID(replica=0, counter=0)


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
    _tasks: task_.Client
    directories: directory.Client
    """Creates, renames, deletes, and moves the directories of Arcs."""

    def __init__(self, client: UnaryClient, tasks: task_.Client) -> None:
        self._client = client
        self._tasks = tasks
        self.directories = directory.Client(client)

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
        directories: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_program: bool = ...,
        include_changes: bool = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Arc: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: str,
        directories: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        include_program: bool = ...,
        include_changes: bool = ...,
        include_status: bool = ...,
        one: bool = ...,
    ) -> Arc: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        name: filter.StringInput = ...,
        directories: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        include_program: bool = ...,
        include_changes: bool = ...,
        include_status: bool = ...,
        one: Literal[True],
    ) -> Arc: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        name: list[str] | filter.StringDict | filter.String = ...,
        directories: UUID | list[UUID] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        names: list[str] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        include_program: bool = ...,
        include_changes: bool = ...,
        include_status: bool = ...,
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
        order: Order | None = None,
        after: Cursor | None = None,
        include_program: bool = False,
        include_changes: bool = False,
        include_status: bool = False,
        one: bool = False,
        **fields: Any,
    ) -> Arc | list[Arc]:
        """Retrieves Arcs from the Core. With no arguments, retrieves every Arc.

        Each filter field takes a value, a list of values, or an operator, and an Arc
        must match every field. A plain ``key`` or ``name`` returns one Arc.

        :param key: Keeps Arcs with this key.
        :param name: Keeps Arcs with this name.
        :param directories: Keeps Arcs in any of these directories.
        :param and_: Keeps Arcs that match every filter in the list.
        :param or_: Keeps Arcs that match any filter in the list.
        :param not_: Keeps Arcs that do not match the filter.
        :param where: A filter tree that the Arcs must also match.
        :param keys: Keeps Arcs with one of these keys.
        :param names: Keeps Arcs with one of these names.
        :param search_term: A fuzzy search term matched against Arc names.
        :param limit: The maximum number of Arcs to retrieve.
        :param order: The order of the Arcs. Defaults to key order.
        :param after: Retrieves the Arcs past this position in the order: the key of the
            last Arc of the page before, and its value of the order field.
        :param include_program: Whether the Core compiles each automation and sets
            its program. The Core raises an error if any automation fails to compile.
        :param include_changes: Whether the Core sets, on each automation, what changed
            since its task was deployed.
        :param include_status: Whether the Core sets each module's compile status.
        :param one: Returns one Arc for any filter.
        :returns: One Arc for a plain ``key`` or ``name`` or when ``one`` is set,
            otherwise a list of Arcs. The list leaves out keys that do not exist.
        :raises NotFoundError: If one Arc is wanted and none matches.
        :raises MultipleFoundError: If one Arc is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one Arc is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(
            key=None if keys is None else filter.Set[Key](in_=keys),
            name=None if names is None else filter.String(in_=names),
        )
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
            include_program=include_program,
            include_changes=include_changes,
            include_status=include_status,
        )
        res = self._client.send("/arc/retrieve", req, RetrieveResponse)
        arcs = self._sugar(res.arcs)
        return sole("Arc", req, arcs) if single else arcs

    def delete(self, keys: Key | list[Key]) -> None:
        self._client.send("/arc/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def set_text(self, key: Key, text: str) -> None:
        """Replaces the text of the text-mode Arc with key. Edits that other editors
        make between the read and the write merge into the result.

        :raises NotFoundError: If no Arc has key.
        :raises ValueError: If the Arc is a graph-mode Arc.
        """
        source = self.retrieve(key=key).source
        if not isinstance(source, TextSource):
            raise ValueError(f"Arc {key} is a graph-mode Arc and has no text")
        doc = source.text.doc
        deleted = {(d.id.replica, d.id.counter) for d in doc.deletes}
        actions: list[actions_gen.Action] = [
            actions_gen.delete_char(actions_gen.DeleteCharPayload(id=i.id))
            for i in doc.inserts
            if (i.id.replica, i.id.counter) not in deleted
        ]
        used = {i.id.replica for i in doc.inserts}
        replica = _SEED_REPLICA
        while replica == _SEED_REPLICA or replica in used:
            replica = random.randint(_SEED_REPLICA + 1, _MAX_REPLICA)
        origin = _ROOT
        for counter, char in enumerate(text):
            id_ = crdt.ID(replica=replica, counter=counter)
            payload = actions_gen.InsertCharPayload(
                id=id_, origin=origin, side="right", char=ord(char)
            )
            actions.append(actions_gen.insert_char(payload))
            origin = id_
        self._client.send(
            "/arc/dispatch", _DispatchRequest(key=key, actions=actions), Empty
        )

    def update_task(
        self,
        key: Key,
        *,
        rack: rack_.Key | None = None,
        performance: Performance | None = None,
    ) -> task_.Task | None:
        """Changes the task of the automation with key. A field left as None stays as
        it is.

        :param key: The key of the automation.
        :param rack: Binds the automation to the rack, creating its task or moving the
            existing one. A rack of 0 unbinds it and deletes its task.
        :param performance: Sets how closely the task's runtime holds timer deadlines.
        :returns: The task, or None after an unbind.
        :raises ValidationError: When unbinding while the task is running, or when
            setting the performance of an automation with no rack.
        """
        res = self._client.send(
            "/arc/update-task",
            _UpdateTaskRequest(key=key, rack=rack, performance=performance),
            _UpdateTaskResponse,
        )
        return None if res.task is None else self._tasks.sugar([res.task])[0]

    def _sugar(self, payloads: list[Payload]) -> list[Arc]:
        return [_ARCS[p.type].model_validate(p, from_attributes=True) for p in payloads]
