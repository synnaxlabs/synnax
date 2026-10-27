#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, overload

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax import project
from synnax.lineplot.types_gen import Key, LinePlot
from x.lists import normalize


class _CreateRequest(BaseModel):
    project: project.Key
    line_plots: list[LinePlot]


class _CreateResponse(BaseModel):
    line_plots: list[LinePlot]


class _RetrieveRequest(BaseModel):
    keys: list[Key]


class _RetrieveResponse(BaseModel):
    line_plots: list[LinePlot] = Field(default_factory=list)


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RenamePayload(BaseModel):
    name: str


class _RenameAction(BaseModel):
    type: Literal["rename"] = "rename"
    rename: _RenamePayload


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[_RenameAction]


class Client:
    """Client for creating, retrieving, renaming, and deleting line plots on a Synnax
    Core. A line plot belongs to exactly one project.
    """

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, *, project: project.Key, name: str) -> LinePlot: ...

    @overload
    def create(self, line_plots: LinePlot, *, project: project.Key) -> LinePlot: ...

    @overload
    def create(
        self, line_plots: list[LinePlot], *, project: project.Key
    ) -> list[LinePlot]: ...

    def create(
        self,
        line_plots: LinePlot | list[LinePlot] | None = None,
        *,
        project: project.Key,
        name: str = "",
    ) -> LinePlot | list[LinePlot]:
        """Create one or more line plots in a project.

        :param line_plots: A single line plot or a list of them. When omitted, a line
            plot is created from the ``name`` keyword argument.
        :param project: The key of the project that owns the line plots.
        :param name: The name of the line plot to create when ``line_plots`` is
            omitted.
        :returns: The created line plot, or a list of created line plots when
            ``line_plots`` is a list.
        """
        is_single = not isinstance(line_plots, list)
        if line_plots is None:
            line_plots = LinePlot(name=name)
        req = _CreateRequest(project=project, line_plots=normalize(line_plots))
        res = self._client.send("/lineplot/create", req, _CreateResponse)
        return res.line_plots[0] if is_single else res.line_plots

    @overload
    def retrieve(self, *, key: Key) -> LinePlot: ...

    @overload
    def retrieve(self, *, keys: list[Key]) -> list[LinePlot]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
    ) -> LinePlot | list[LinePlot]:
        """Retrieve one or more line plots by key.

        :param key: The key of a single line plot to retrieve.
        :param keys: The keys of the line plots to retrieve.
        :returns: The matching line plot when ``key`` is provided, otherwise a list of
            matching line plots.
        :raises NotFoundError: If any of the requested line plots does not exist.
        """
        res = self._client.send(
            "/lineplot/retrieve",
            _RetrieveRequest(keys=normalize(key, keys)),
            _RetrieveResponse,
        )
        return res.line_plots[0] if key is not None else res.line_plots

    def rename(self, key: Key, name: str) -> None:
        """Rename a line plot.

        :param key: The key of the line plot to rename.
        :param name: The new name for the line plot.
        """
        req = _DispatchRequest(
            key=key,
            actions=[_RenameAction(rename=_RenamePayload(name=name))],
        )
        self._client.send("/lineplot/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more line plots by key.

        :param keys: A single line plot key or a list of keys to delete.
        """
        self._client.send(
            "/lineplot/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )
