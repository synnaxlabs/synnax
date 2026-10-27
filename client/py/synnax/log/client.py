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
from synnax import project as project_
from synnax.log.types_gen import Key, Log
from x.lists import normalize


class _CreateRequest(BaseModel):
    project: project_.Key
    logs: list[Log]


class _CreateResponse(BaseModel):
    logs: list[Log]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RetrieveRequest(BaseModel):
    keys: list[Key]


class _RetrieveResponse(BaseModel):
    logs: list[Log] = Field(default_factory=list)


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
    """Client for creating, retrieving, renaming, and deleting logs in a Synnax Core.

    A log is a Console visualization that displays the samples of a set of channels as
    timestamped rows.
    """

    _client: UnaryClient

    def __init__(self, client: UnaryClient) -> None:
        self._client = client

    @overload
    def create(self, project: project_.Key, *, name: str) -> Log: ...

    @overload
    def create(self, project: project_.Key, logs: Log) -> Log: ...

    @overload
    def create(self, project: project_.Key, logs: list[Log]) -> list[Log]: ...

    def create(
        self,
        project: project_.Key,
        logs: Log | list[Log] | None = None,
        *,
        name: str = "",
    ) -> Log | list[Log]:
        """Create one or more logs in a project.

        :param project: The key of the project that owns the logs.
        :param logs: A single log or a list of them. When omitted, an empty log is
            created from the ``name`` keyword argument.
        :param name: The name of the log to create when ``logs`` is omitted.
        :returns: The created log, or a list of created logs when ``logs`` is a list.
        """
        is_single = not isinstance(logs, list)
        if logs is None:
            logs = [Log(name=name)]
        req = _CreateRequest(project=project, logs=normalize(logs))
        res = self._client.send("/log/create", req, _CreateResponse)
        return res.logs[0] if is_single else res.logs

    @overload
    def retrieve(self, *, key: Key) -> Log: ...

    @overload
    def retrieve(self, *, keys: list[Key]) -> list[Log]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
    ) -> Log | list[Log]:
        """Retrieve one or more logs by key.

        :param key: The key of a single log to retrieve.
        :param keys: The keys of the logs to retrieve.
        :returns: The log when ``key`` is provided, otherwise a list of logs.
        :raises NotFoundError: If any of the requested logs does not exist.
        """
        is_single = key is not None
        req = _RetrieveRequest(keys=normalize(key, keys))
        res = self._client.send("/log/retrieve", req, _RetrieveResponse)
        return res.logs[0] if is_single else res.logs

    def rename(self, key: Key, name: str) -> None:
        """Rename a log.

        :param key: The key of the log to rename.
        :param name: The new name for the log.
        """
        action = _RenameAction(rename=_RenamePayload(name=name))
        req = _DispatchRequest(key=key, actions=[action])
        self._client.send("/log/dispatch", req, Empty)

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more logs by key.

        :param keys: A single log key or a list of keys to delete.
        """
        self._client.send("/log/delete", _DeleteRequest(keys=normalize(keys)), Empty)
