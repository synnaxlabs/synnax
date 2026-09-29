#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import overload

from pydantic import BaseModel, Field

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.exceptions import NotFoundError
from synnax.library.types_gen import Entry, Key, Library
from x.lists import normalize


class _CreateRequest(BaseModel):
    libraries: list[Library]


_CreateResponse = _CreateRequest


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    search_term: str | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveResponse(BaseModel):
    libraries: list[Library] = Field(default_factory=list)


class Client:
    """Client for creating, retrieving, and deleting libraries on a Synnax cluster.

    A library is a named container of typed entries, such as enums and bus message
    layouts, that tasks reference by key.
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
        entries: list[Entry] | None = None,
    ) -> Library: ...

    @overload
    def create(self, libraries: Library) -> Library: ...

    @overload
    def create(self, libraries: list[Library]) -> list[Library]: ...

    def create(
        self,
        libraries: Library | list[Library] | None = None,
        *,
        name: str = "",
        entries: list[Entry] | None = None,
    ) -> Library | list[Library]:
        """Create one or more libraries, or replace those whose keys already exist.

        :param libraries: A single library or a list of them. When omitted, a library
            is created from the ``name`` and ``entries`` keyword arguments.
        :param name: The name of the library to create when ``libraries`` is omitted.
        :param entries: The entries of the library to create when ``libraries`` is
            omitted.
        :returns: The created library, or a list of created libraries when
            ``libraries`` is a list.
        :raises ValidationError: If a library or one of its entries is invalid.
        """
        is_single = not isinstance(libraries, list)
        if libraries is None:
            libraries = [
                Library(name=name, entries=entries if entries is not None else [])
            ]
        req = _CreateRequest(libraries=normalize(libraries))
        res = self._client.send("/library/create", req, _CreateResponse)
        return res.libraries[0] if is_single else res.libraries

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more libraries by key.

        :param keys: A single library key or a list of keys to delete.
        :raises ValidationError: If a task uses one of the libraries.
        """
        self._client.send(
            "/library/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    def rename(self, key: Key, name: str) -> None:
        """Rename a library.

        :param key: The key of the library to rename.
        :param name: The new name for the library.
        """
        self._client.send("/library/rename", _RenameRequest(key=key, name=name), Empty)

    @overload
    def retrieve(self, *, key: Key) -> Library: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[Library]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> Library | list[Library]:
        """Retrieve one or more libraries.

        :param key: The key of a single library to retrieve. When provided, a single
            library is returned.
        :param keys: The keys of the libraries to retrieve.
        :param search_term: A fuzzy search term matched against library names.
        :param limit: The maximum number of libraries to return.
        :param offset: The number of libraries to skip before returning results.
        :returns: The matching library when ``key`` is provided, otherwise a list of
            matching libraries.
        :raises NotFoundError: If a requested key has no library.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = self._client.send(
            "/library/retrieve",
            _RetrieveRequest(
                keys=keys,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            _RetrieveResponse,
        )
        libraries = res.libraries
        if is_single:
            if len(libraries) == 0:
                raise NotFoundError("Library not found")
            return libraries[0]
        return libraries
