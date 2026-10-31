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

from alamos import NOOP, Instrumentation
from freighter import Empty, FileTransport, UnaryClient
from synnax import imex
from synnax.project.types_gen import (
    Key,
    Project,
    RetrieveRequest,
    RetrieveResponse,
    Where,
)
from synnax.util import one
from x import filter
from x.fs import FilePath
from x.lists import normalize


class _CreateRequest(BaseModel):
    projects: list[Project]


class _CreateResponse(BaseModel):
    projects: list[Project]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RenameRequest(BaseModel):
    key: Key
    name: str


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
        self._client.send("/project/rename", _RenameRequest(key=key, name=name), Empty)

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
    def retrieve(self, *, key: Key) -> Project: ...

    @overload
    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> list[Project]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        offset: int = 0,
    ) -> Project | list[Project]:
        """Retrieve one or more projects.

        :param key: The key of a single project to retrieve. When provided, a single
            project is returned and NotFoundError is raised if it does not exist.
        :param where: Keeps projects that match this filter tree.
        :param keys: The keys of the projects to retrieve.
        :param search_term: A fuzzy search term matched against project names.
        :param limit: The maximum number of projects to return.
        :param offset: The number of projects to skip before returning results.
        :returns: The matching project when ``key`` is provided, otherwise a list of
            matching projects.
        :raises NotFoundError: If ``key`` is provided and no project exists for it.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        res = self._client.send(
            "/project/retrieve",
            RetrieveRequest(
                where=filter.merge(where, flat),
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            RetrieveResponse,
        )
        projects = res.projects
        if is_single:
            return one("Project", key, projects)
        return projects
