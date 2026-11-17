#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal, overload

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax import group as group_, project as project_
from synnax.log.types_gen import (
    Key,
    Log,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    logs: list[Log]


class _CreateResponse(BaseModel):
    logs: list[Log]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _SetPayload(BaseModel):
    path: list[str]
    value: Any


class _SetAction(BaseModel):
    type: Literal["set"] = "set"
    set: _SetPayload


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[_SetAction]


class Client:
    """Client for creating, retrieving, renaming, and deleting logs in a Synnax Core.

    A log is a Console visualization that displays the samples of a set of channels as
    timestamped rows.
    """

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, name: str, project: project_.Key) -> Log: ...

    @overload
    def create(self, logs: Log) -> Log: ...

    @overload
    def create(self, logs: list[Log]) -> list[Log]: ...

    def create(
        self,
        logs: Log | list[Log] | None = None,
        *,
        name: str = "",
        project: project_.Key | None = None,
    ) -> Log | list[Log]:
        """Create one or more logs.

        :param logs: A single log or a list of them. Each one belongs to a project.
            When omitted, a log is created from the ``name`` and ``project`` keyword
            arguments.
        :param name: The name of the log to create when ``logs`` is omitted.
        :param project: The key of the project that holds the log when ``logs`` is
            omitted.
        :returns: The created log, or a list of created logs when ``logs`` is a list.
        :raises ValidationError: If a log does not belong to a project.
        """
        is_single = not isinstance(logs, list)
        if logs is None:
            logs = Log(name=name, project=project)
        req = _CreateRequest(logs=normalize(logs))
        res = self._client.send("/log/create", req, _CreateResponse)
        return res.logs[0] if is_single else res.logs

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        projects: project_.Key | list[project_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        one: bool = ...,
    ) -> Log: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        one: Literal[True],
    ) -> Log: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        projects: project_.Key | list[project_.Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        one: Literal[False] = ...,
    ) -> list[Log]: ...

    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Log | list[Log]:
        """Retrieves logs. With no arguments, retrieves every log.

        Each filter field takes a value, a list of values, or an operator, and a log
        must match every field. A plain ``key`` returns one log.

        :param key: Keeps logs with this key.
        :param and_: Keeps logs that match every filter in the list.
        :param or_: Keeps logs that match any filter in the list.
        :param not_: Keeps logs that do not match the filter.
        :param where: A filter tree that the logs must also match.
        :param keys: Keeps logs with one of these keys.
        :param projects: Keeps the logs owned by any of these projects.
        :param groups: Keeps logs in any of these groups.
        :param grouped: Keeps only logs in a group when True, and only logs in no group
            when False.
        :param one: Returns one log for any filter.
        :returns: One log for a plain ``key`` or when ``one`` is set, otherwise a list
            of logs. The list leaves out keys that do not exist.
        :raises NotFoundError: If one log is wanted and none matches.
        :raises MultipleFoundError: If one log is wanted and more than one matches.
        :raises ValidationError: If a filter field is None.
        """
        single = returns_one(fields, one)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields)
        )
        res = self._client.send("/log/retrieve", req, RetrieveResponse)
        return sole("log", req, res.logs) if single else res.logs

    def rename(self, key: Key, name: str) -> None:
        """Rename a log.

        :param key: The key of the log to rename.
        :param name: The new name for the log.
        """
        action = _SetAction(set=_SetPayload(path=["name"], value=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/log/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more logs by key.

        :param keys: A single log key or a list of keys to delete.
        """
        self._client.send("/log/delete", _DeleteRequest(keys=normalize(keys)), Empty)
