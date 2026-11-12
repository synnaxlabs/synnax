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

from alamos import NOOP, Instrumentation
from freighter import Empty, FileTransport, UnaryClient
from synnax import group as group_, imex
from synnax.project import actions_gen
from synnax.project.types_gen import (
    Cursor,
    Key,
    Order,
    Project,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
from x.fs import FilePath
from x.lists import normalize


class _CreateRequest(BaseModel):
    projects: list[Project]


class _CreateResponse(BaseModel):
    projects: list[Project]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _ExportRequest(BaseModel):
    key: Key
    encoding: imex.Encoding


class _ImportResponse(BaseModel):
    project: Project


class Client:
    """Client for creating, retrieving, deleting, exporting, and importing projects on
    a Synnax cluster.

    A project is a named, persistable container that groups the panels of the Console
    application.
    """

    _client: UnaryClient
    _file_transport: FileTransport
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        *,
        client: UnaryClient,
        file_transport: FileTransport,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self._file_transport = file_transport
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        name: str,
    ) -> Project: ...

    @overload
    def create(self, projects: Project) -> Project: ...

    @overload
    def create(self, projects: list[Project]) -> list[Project]: ...

    def create(
        self,
        projects: Project | list[Project] | None = None,
        *,
        name: str = "",
    ) -> Project | list[Project]:
        """Create one or more projects.

        :param projects: A single project or a list of them. When omitted, a
            project is created from the ``name`` keyword argument.
        :param name: The name of the project to create when ``projects`` is omitted.
        :returns: The created project, or a list of created projects when ``projects``
            is a list.
        """
        is_single = not isinstance(projects, list)
        if projects is None:
            projects = [Project(name=name)]
        req = _CreateRequest(projects=normalize(projects))
        res = self._client.send("/project/create", req, _CreateResponse)
        return res.projects[0] if is_single else res.projects

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more projects by key.

        :param keys: A single project key or a list of keys to delete.
        """
        self._client.send(
            "/project/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    def rename(self, key: Key, name: str) -> None:
        """Rename a project.

        :param key: The key of the project to rename.
        :param name: The new name for the project.
        """
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/project/dispatch", req, Empty)

    def export(
        self, key: Key, dest: FilePath, *, encoding: imex.Encoding = "JSON"
    ) -> None:
        """Exports a project and its contents as a bundle, streaming it into dest.

        The bundle is a zip archive with one file per document and panel, group
        children as directories, and a manifest.json that names the project.

        :param key: The key of the project to export.
        :param dest: The path to write the bundle to. It must have a ``.zip``
            extension. The write is atomic: a failure leaves an existing dest as it was.
        :param encoding: The serialization of the member files.
        :raises NotFoundError: If no project exists for key.
        """
        self._file_transport.download(
            "/project/export", _ExportRequest(key=key, encoding=encoding), dest=dest
        )

    def import_(self, source: FilePath) -> Project:
        """Imports a project bundle from a zip archive, streamed from disk.

        The Core creates a new project and imports every member in one transaction,
        so a failure leaves nothing behind. When the manifest has no name, the Core
        names the project after the file, with the extension removed.

        :param source: The path of the bundle. It must have a ``.zip`` extension.
        :returns: The created project.
        :raises ValidationError: If the manifest is missing, malformed, or of another
            bundle kind, if two member names collide, or if a member cannot be
            imported.
        """
        return self._file_transport.upload(
            "/project/import", source, _ImportResponse
        ).project

    @overload
    def retrieve(
        self,
        *,
        key: Key,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> Project: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> Project: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        groups: group_.Key | list[group_.Key] = ...,
        grouped: bool = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        order: Order | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[Project]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        order: Order | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> Project | list[Project]:
        """Retrieves projects. With no arguments, retrieves every project.

        Each filter field takes a value, a list of values, or an operator, and a project
        must match every field. A plain ``key`` returns one project.

        :param key: Keeps projects with this key.
        :param groups: Keeps projects in any of these groups.
        :param grouped: Keeps only projects in a group when True, and only projects in
            no group when False.
        :param and_: Keeps projects that match every filter in the list.
        :param or_: Keeps projects that match any filter in the list.
        :param not_: Keeps projects that do not match the filter.
        :param where: A filter tree that the projects must also match.
        :param keys: Keeps projects with one of these keys.
        :param search_term: A fuzzy search term matched against project names.
        :param limit: The maximum number of projects to retrieve.
        :param order: The order of the projects. Defaults to key order.
        :param after: Retrieves the projects past this position in the order: the key of
            the last project of the page before, and its value of the order field.
        :param one: Returns one project for any filter.
        :returns: One project for a plain ``key`` or when ``one`` is set, otherwise a
            list of projects. The list leaves out keys that do not exist.
        :raises NotFoundError: If one project is wanted and none matches.
        :raises MultipleFoundError: If one project is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one project is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            order=order,
            after=after,
        )
        res = self._client.send("/project/retrieve", req, RetrieveResponse)
        return sole("project", req, res.projects) if single else res.projects
