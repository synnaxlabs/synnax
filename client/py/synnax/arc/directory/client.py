#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pydantic import BaseModel, Field

from freighter import Empty, UnaryClient
from synnax.arc.directory.types_gen import (
    Directory,
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Where,
)
from synnax.resource import ID, CrudeID
from x.lists import normalize


class _Directory(BaseModel):
    key: Key | None = None
    name: str


class _CreateRequest(BaseModel):
    parent: Key
    directory: _Directory


class _CreateResponse(BaseModel):
    directory: Directory


class _RenameRequest(BaseModel):
    key: Key
    name: str


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _MoveRequest(BaseModel):
    from_: Key = Field(alias="from")
    to: Key
    ids: list[ID]


class Client:
    """Creates, renames, deletes, and moves the directories that hold Arcs."""

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    def root(self) -> Key:
        """Returns the key of the Arcs root, the directory every Arc starts in.

        :raises LookupError: If the Core has no Arcs root.
        """
        res = self._client.send(
            "/arc/directory/retrieve",
            RetrieveRequest(where=Where(nested=False)),
            RetrieveResponse,
        )
        if not res.directories:
            raise LookupError("the Core has no Arcs root")
        return res.directories[0].key

    def create(self, name: str, *, parent: Key, key: Key | None = None) -> Directory:
        """Creates a directory in the directory parent.

        :param name: The name of the new directory, one segment of an import path.
        :param parent: The key of the directory to create the new directory in.
        :param key: The key of the new directory. The Core generates one when absent.
        :returns: The created directory with its key populated.
        """
        return self._client.send(
            "/arc/directory/create",
            _CreateRequest(parent=parent, directory=_Directory(key=key, name=name)),
            _CreateResponse,
        ).directory

    def rename(self, key: Key, name: str) -> None:
        """Renames the directory with key."""
        self._client.send(
            "/arc/directory/rename", _RenameRequest(key=key, name=name), Empty
        )

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes the empty directories with keys."""
        self._client.send(
            "/arc/directory/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    def move(self, from_: Key, to: Key, *children: CrudeID) -> None:
        """Moves the Arcs and directories children from the directory from_ into the
        directory to.
        """
        self._client.send(
            "/arc/move",
            _MoveRequest.model_validate(
                {"from": from_, "to": to, "ids": [ID(c) for c in children]}
            ),
            Empty,
        )
