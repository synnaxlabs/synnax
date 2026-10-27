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
from synnax.exceptions import NotFoundError
from synnax.log.types_gen import Key, Log
from x.lists import normalize


class _CreateRequest(BaseModel):
    project: project.Key
    logs: list[Log]


class _CreateResponse(BaseModel):
    logs: list[Log]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _RetrieveRequest(BaseModel):
    keys: list[Key]
    ignore_not_found_error: bool = False


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
    def create(self, project: project.Key, logs: Log) -> Log: ...

    @overload
    def create(self, project: project.Key, logs: list[Log]) -> list[Log]: ...

    def create(self, project: project.Key, logs: Log | list[Log]) -> Log | list[Log]:
        """Create one or more logs in a project.

        :param project: The key of the project that owns the logs.
        :param logs: A single log or a list of them.
        :returns: The created log, or a list of created logs when ``logs`` is a list.
        """
        req = _CreateRequest(project=project, logs=normalize(logs))
        res = self._client.send("/log/create", req, _CreateResponse)
        return res.logs if isinstance(logs, list) else res.logs[0]

    @overload
    def retrieve(self, *, key: Key) -> Log: ...

    @overload
    def retrieve(
        self, *, keys: list[Key], ignore_not_found_error: bool = False
    ) -> list[Log]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        ignore_not_found_error: bool = False,
    ) -> Log | list[Log]:
        """Retrieve one or more logs by key.

        :param key: The key of a single log to retrieve.
        :param keys: The keys of the logs to retrieve.
        :param ignore_not_found_error: When true, omit missing logs from the result
            instead of raising NotFoundError. Only applies to ``keys``.
        :returns: The log when ``key`` is provided, otherwise a list of logs.
        :raises NotFoundError: If a requested log does not exist.
        """
        if key is not None:
            res = self._client.send(
                "/log/retrieve", _RetrieveRequest(keys=[key]), _RetrieveResponse
            )
            if len(res.logs) == 0:
                raise NotFoundError("Log not found")
            return res.logs[0]
        res = self._client.send(
            "/log/retrieve",
            _RetrieveRequest(
                keys=keys or [], ignore_not_found_error=ignore_not_found_error
            ),
            _RetrieveResponse,
        )
        return res.logs

    def rename(self, key: Key, name: str) -> None:
        """Rename a log.

        :param key: The key of the log to rename.
        :param name: The new name for the log.
        """
        action = _RenameAction(rename=_RenamePayload(name=name))
        self._client.send(
            "/log/dispatch", _DispatchRequest(key=key, actions=[action]), Empty
        )

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more logs by key. Deleting a missing log is not an error.

        :param keys: A single log key or a list of keys to delete.
        """
        self._client.send("/log/delete", _DeleteRequest(keys=normalize(keys)), Empty)
