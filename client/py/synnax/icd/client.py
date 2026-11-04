#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, overload

from pydantic import BaseModel, ConfigDict, Field

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.exceptions import NotFoundError
from synnax.icd.types_gen import ICD, Key, Message
from x.lists import normalize


class Problem(BaseModel):
    """A problem that a check finds in an ICD."""

    path: str = ""
    """The dotted location of the bad value, such as "messages.0.name". It is empty for
    a problem with the whole ICD."""
    message: str
    """What is wrong."""


class _CreateRequest(BaseModel):
    icds: list[ICD]


_CreateResponse = _CreateRequest


class _CheckRequest(BaseModel):
    icd: ICD


class _CheckResponse(BaseModel):
    problems: list[Problem]


class _DeleteRequest(BaseModel):
    keys: list[Key]


class _SetPayload(BaseModel):
    path: list[str]
    value: str


class _SetAction(BaseModel):
    type: Literal["set"] = "set"
    set: _SetPayload


class _DispatchRequest(BaseModel):
    key: Key
    dispatch_key: str = ""
    actions: list[_SetAction]


ImportFormat = Literal["dbc", "csv", "xlsx"]
"""The format of an interface control document file."""


class _ImportRequest(BaseModel):
    model_config = ConfigDict(ser_json_bytes="base64")
    key: Key
    format: ImportFormat
    data: bytes


class _ImportResponse(BaseModel):
    icd: ICD


class _RetrieveRequest(BaseModel):
    keys: list[Key] | None = None
    search_term: str | None = None
    limit: int | None = None
    offset: int | None = None


class _RetrieveResponse(BaseModel):
    icds: list[ICD] = Field(default_factory=list)


class Client:
    """Client for creating, checking, retrieving, and deleting ICDs on a Synnax cluster.

    An ICD (interface control document) holds the messages a device sends and receives
    on a bus or byte stream. Tasks reference an ICD by key.
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
        messages: list[Message] | None = None,
    ) -> ICD: ...

    @overload
    def create(self, icds: ICD) -> ICD: ...

    @overload
    def create(self, icds: list[ICD]) -> list[ICD]: ...

    def create(
        self,
        icds: ICD | list[ICD] | None = None,
        *,
        name: str = "",
        messages: list[Message] | None = None,
    ) -> ICD | list[ICD]:
        """Create one or more ICDs, or replace whole those whose keys already exist.

        :param icds: A single ICD or a list of them. When omitted, an ICD is created
            from the ``name`` and ``messages`` keyword arguments.
        :param name: The name of the ICD to create when ``icds`` is omitted.
        :param messages: The messages of the ICD to create when ``icds`` is omitted.
        :returns: The created ICD, or a list of created ICDs when ``icds`` is a list.
        :raises ValidationError: If an ICD or one of its messages breaks a rule of its
            schema. Problems that ``check`` finds do not stop a save.
        """
        is_single = not isinstance(icds, list)
        if icds is None:
            icds = [ICD(name=name, messages=messages if messages is not None else [])]
        req = _CreateRequest(icds=normalize(icds))
        res = self._client.send("/icd/create", req, _CreateResponse)
        return res.icds[0] if is_single else res.icds

    def check(self, icd: ICD) -> list[Problem]:
        """Check an ICD without saving it.

        :param icd: The ICD to check.
        :returns: Every problem in the ICD, or an empty list.
        """
        res = self._client.send("/icd/check", _CheckRequest(icd=icd), _CheckResponse)
        return res.problems

    def delete(self, keys: Key | list[Key]) -> None:
        """Delete one or more ICDs by key.

        :param keys: A single ICD key or a list of keys to delete.
        """
        self._client.send("/icd/delete", _DeleteRequest(keys=normalize(keys)), Empty)

    def rename(self, key: Key, name: str) -> None:
        """Rename an ICD.

        :param key: The key of the ICD to rename.
        :param name: The new name for the ICD.
        """
        req = _DispatchRequest(
            key=key,
            actions=[_SetAction(set=_SetPayload(path=["name"], value=name))],
        )
        self._client.send("/icd/dispatch", req, Empty)

    def import_file(self, key: Key, data: bytes, format: ImportFormat) -> ICD:
        """Import an interface control document file into an ICD. Messages and fields
        that match existing ones by name keep their keys.

        :param key: The key of the ICD to import into.
        :param data: The contents of the file.
        :param format: The format of the file: "dbc", "csv", or "xlsx".
        :returns: The ICD after the import.
        :raises ValidationError: If the file cannot be parsed or the result is invalid.
        """
        req = _ImportRequest(key=key, format=format, data=data)
        return self._client.send("/icd/import", req, _ImportResponse).icd

    @overload
    def retrieve(self, *, key: Key) -> ICD: ...

    @overload
    def retrieve(
        self,
        *,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[ICD]: ...

    def retrieve(
        self,
        *,
        key: Key | None = None,
        keys: list[Key] | None = None,
        search_term: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> ICD | list[ICD]:
        """Retrieve one or more ICDs.

        :param key: The key of a single ICD to retrieve. When provided, a single ICD is
            returned.
        :param keys: The keys of the ICDs to retrieve.
        :param search_term: A fuzzy search term matched against ICD names.
        :param limit: The maximum number of ICDs to return.
        :param offset: The number of ICDs to skip before returning results.
        :returns: The matching ICD when ``key`` is provided, otherwise a list of
            matching ICDs.
        :raises NotFoundError: If a requested key has no ICD.
        """
        is_single = key is not None
        if key is not None:
            keys = [key]
        res = self._client.send(
            "/icd/retrieve",
            _RetrieveRequest(
                keys=keys,
                search_term=search_term,
                limit=limit,
                offset=offset,
            ),
            _RetrieveResponse,
        )
        if is_single:
            if len(res.icds) == 0:
                raise NotFoundError("ICD not found")
            return res.icds[0]
        return res.icds
