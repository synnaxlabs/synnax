#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, overload
from uuid import UUID

from pydantic import BaseModel

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.exceptions import NotFoundError
from synnax.project.types_gen import Key, New, Project
from x.normalize import normalize


class _CreateRequest(BaseModel):
    projects: list[New]


class _CreateResponse(BaseModel):
    projects: list[Project]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    search_term: str | None = None
    author: UUID | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveResponse(BaseModel):
    projects: list[Project] | None = None


class Client:
    """Client for creating, retrieving, and deleting projects on a Synnax cluster.

    A project is a named, persistable container storing the layout of the Console
    application.
    """

    _client: UnaryClient
    instrumentation: Instrumentation = NOOP

    def __init__(
        self,
        client: UnaryClient,
        instrumentation: Instrumentation = NOOP,
    ) -> None:
        self._client = client
        self.instrumentation = instrumentation

    @overload
    def create(
        self,
        *,
        name: str,
        layout: dict[str, Any] | None = None,
    ) -> Project: ...

    @overload
    def create(self, projects: New) -> Project: ...

    @overload
    def create(self, projects: list[New]) -> list[Project]: ...

    def create(
        self,
        projects: New | list[New] | None = None,
        *,
        name: str = "",
        layout: dict[str, Any] | None = None,
    ) -> Project | list[Project]:
        """Create one or more projects.

        :param projects: A single New project or a list of them. When omitted, a
            project is created from the ``name`` and ``layout`` keyword arguments.
        :param name: The name of the project to create when ``projects`` is omitted.
        :param layout: The initial layout of the project when ``projects`` is omitted.
            Defaults to an empty layout.
        :returns: The created project, or a list of created projects when ``projects``
            is a list.
        """
        is_single = not isinstance(projects, list)
        if projects is None:
            projects = [New(name=name, layout=layout if layout is not None else {})]
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

    @overload
    def retrieve(self, *, key: Key) -> Project: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        author: UUID | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[Project]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        author: UUID | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> Project | list[Project]:
        """Retrieve one or more projects.

        :param key: The key of a single project to retrieve. When provided, a single
            project is returned and NotFoundError is raised if it does not exist.
        :param keys: The keys of the projects to retrieve.
        :param search_term: A fuzzy search term matched against project names.
        :param author: Restrict results to projects created by this user.
        :param limit: The maximum number of projects to return.
        :param offset: The number of projects to skip before returning results.
        :returns: The matching project when ``key`` is provided, otherwise a list of
            matching projects.
        :raises NotFoundError: If ``key`` is provided and no project exists for it.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = self._client.send(
            "/project/retrieve",
            _RetrieveRequest(
                keys=keys,
                search_term=search_term,
                author=author,
                limit=limit,
                offset=offset,
            ),
            _RetrieveResponse,
        )
        projects = res.projects if res.projects is not None else []
        if is_single:
            if len(projects) == 0:
                raise NotFoundError("Project not found")
            return projects[0]
        return projects
