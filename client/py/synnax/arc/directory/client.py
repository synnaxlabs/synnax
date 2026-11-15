#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from uuid import UUID

from pydantic import BaseModel

from freighter import Empty, UnaryClient
from synnax.arc import actions_gen as arc_actions
from synnax.arc.directory import actions_gen
from synnax.arc.directory.types_gen import (
    Directory,
    Key,
    RetrieveRequest,
    RetrieveResponse,
    Where,
)
from synnax.resource import ID, CrudeID
from synnax.transport import send_requests
from x.lists import normalize


class _Directory(BaseModel):
    key: Key | None = None
    name: str


class _CreateRequest(BaseModel):
    parent: Key
    directory: _Directory


class _CreateResponse(BaseModel):
    directory: Directory


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[actions_gen.Action]


class _ArcDispatchRequest(BaseModel):
    key: UUID
    dispatch_key: str = ""
    actions: list[arc_actions.Action]


class _DeleteRequest(BaseModel):
    keys: list[Key]


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
        action = actions_gen.set(actions_gen.SetPayload(path=["name"], value=name))
        self._client.send(
            "/arc/directory/dispatch",
            _DispatchRequest(key=key, actions=[action]),
            Empty,
        )

    def delete(self, keys: Key | list[Key]) -> None:
        """Deletes the empty directories with keys."""
        self._client.send(
            "/arc/directory/delete", _DeleteRequest(keys=normalize(keys)), Empty
        )

    def move(self, to: Key, *children: CrudeID) -> None:
        """Moves the Arcs and directories children into the directory to, in one
        transaction.
        """
        requests: list[tuple[str, BaseModel]] = []
        for child in children:
            id_ = ID(child)
            key = UUID(id_.key)
            if id_.type == "arc":
                payload = arc_actions.SetDirectoryPayload(directory=to)
                arc_req = _ArcDispatchRequest(
                    key=key, actions=[arc_actions.set_directory(payload)]
                )
                requests.append(("/arc/dispatch", arc_req))
            else:
                set_ = actions_gen.SetPayload(path=["parent"], value=str(to))
                req = _DispatchRequest(key=key, actions=[actions_gen.set(set_)])
                requests.append(("/arc/directory/dispatch", req))
        send_requests(self._client, requests)
