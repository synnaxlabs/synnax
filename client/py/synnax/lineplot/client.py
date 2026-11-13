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
from synnax.lineplot.types_gen import (
    Key,
    LinePlot,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.lists import normalize


class _CreateRequest(BaseModel):
    line_plots: list[LinePlot]


class _CreateResponse(BaseModel):
    line_plots: list[LinePlot]


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
    """Client for creating, retrieving, renaming, and deleting line plots on a Synnax
    Core. A line plot belongs to a project.
    """

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, name: str, project: project_.Key) -> LinePlot: ...

    @overload
    def create(self, line_plots: LinePlot) -> LinePlot: ...

    @overload
    def create(self, line_plots: list[LinePlot]) -> list[LinePlot]: ...

    def create(
        self,
        line_plots: LinePlot | list[LinePlot] | None = None,
        *,
        name: str = "",
        project: project_.Key | None = None,
    ) -> LinePlot | list[LinePlot]:
        """Create one or more line plots.

        :param line_plots: A single line plot or a list of them. Each one belongs to a
            project. When omitted, a line plot is created from the ``name`` and
            ``project`` keyword arguments.
        :param name: The name of the line plot to create when ``line_plots`` is omitted.
        :param project: The key of the project that owns the line plot when
            ``line_plots`` is omitted.
        :returns: The created line plot, or a list of created line plots when
            ``line_plots`` is a list.
        :raises ValidationError: If a line plot does not belong to a project.
        """
        is_single = not isinstance(line_plots, list)
        if line_plots is None:
            line_plots = LinePlot(name=name, project=project)
        req = _CreateRequest(line_plots=normalize(line_plots))
        res = self._client.send("/lineplot/create", req, _CreateResponse)
        return res.line_plots[0] if is_single else res.line_plots

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
    ) -> LinePlot: ...

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
    ) -> LinePlot: ...

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
    ) -> list[LinePlot]: ...

    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        where: Where | None = None,
        one: bool = False,
        **fields: Any,
    ) -> LinePlot | list[LinePlot]:
        """Retrieves line plots. With no arguments, retrieves every line plot.

        Each filter field takes a value, a list of values, or an operator, and a line
        plot must match every field. A plain ``key`` returns one line plot.

        :param key: Keeps line plots with this key.
        :param and_: Keeps line plots that match every filter in the list.
        :param or_: Keeps line plots that match any filter in the list.
        :param not_: Keeps line plots that do not match the filter.
        :param where: A filter tree that the line plots must also match.
        :param keys: Keeps line plots with one of these keys.
        :param projects: Keeps the line plots owned by any of these projects.
        :param groups: Keeps line plots in any of these groups.
        :param grouped: Keeps only line plots in a group when True, and only line plots
            in no group when False.
        :param one: Returns one line plot for any filter.
        :returns: One line plot for a plain ``key`` or when ``one`` is set, otherwise a
            list of line plots. The list leaves out keys that do not exist.
        :raises NotFoundError: If one line plot is wanted and none matches.
        :raises MultipleFoundError: If one line plot is wanted and more than one
            matches.
        :raises ValidationError: If a filter field is None.
        """
        single = returns_one(fields, one)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields)
        )
        res = self._client.send("/lineplot/retrieve", req, RetrieveResponse)
        return sole("line plot", req, res.line_plots) if single else res.line_plots

    def rename(self, key: Key, name: str) -> None:
        """Rename a line plot.

        :param key: The key of the line plot to rename.
        :param name: The new name for the line plot.
        """
        req = _DispatchRequest(
            key=key,
            actions=[_SetAction(set=_SetPayload(path=["name"], value=name))],
        )
        self._client.send("/lineplot/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more line plots by key.

        :param keys: A single line plot key or a list of keys to delete.
        """
        self._client.send(
            "/lineplot/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )
