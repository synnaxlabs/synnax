#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
from typing import Any, Literal, overload

from pydantic import BaseModel, field_serializer

from alamos import NOOP, Instrumentation
from freighter import Empty, UnaryClient
from synnax.icd.types_gen import (
    ICD,
    Cursor,
    Key,
    Message,
    RetrieveRequest,
    RetrieveResponse,
    Where,
    WhereInput,
)
from synnax.util import request_where, returns_one, sole
from x import filter
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
    key: Key
    format: ImportFormat
    data: bytes

    # Pydantic's base64 is the URL-safe alphabet, which the Core does not decode.
    @field_serializer("data")
    def _serialize_data(self, data: bytes) -> str:
        return base64.b64encode(data).decode()


class _ImportResponse(BaseModel):
    icd: ICD


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
        """Create one or more ICDs.

        :param icds: A single ICD or a list of them. When omitted, an ICD is created
            from the ``name`` and ``messages`` keyword arguments.
        :param name: The name of the ICD to create when ``icds`` is omitted.
        :param messages: The messages of the ICD to create when ``icds`` is omitted.
        :returns: The created ICD, or a list of created ICDs when ``icds`` is a list.
        :raises ValidationError: If an ICD with the same key exists, or if an ICD or one
            of its messages breaks a rule of its schema. Problems that ``check`` finds
            do not stop a save.
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
    def retrieve(
        self,
        *,
        key: Key,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        search_term: str = ...,
        one: bool = ...,
    ) -> ICD: ...

    @overload
    def retrieve(
        self,
        *,
        key: filter.SetInput[Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        one: Literal[True],
    ) -> ICD: ...

    @overload
    def retrieve(
        self,
        *,
        key: list[Key] | filter.SetDict[Key] | filter.Set[Key] = ...,
        and_: list[WhereInput] = ...,
        or_: list[WhereInput] = ...,
        not_: WhereInput = ...,
        where: Where | None = ...,
        keys: list[Key] | None = ...,
        search_term: str = ...,
        limit: int | None = ...,
        after: Cursor | None = ...,
        one: Literal[False] = ...,
    ) -> list[ICD]: ...

    def retrieve(
        self,
        *,
        where: Where | None = None,
        keys: list[Key] | None = None,
        search_term: str = "",
        limit: int | None = None,
        after: Cursor | None = None,
        one: bool = False,
        **fields: Any,
    ) -> ICD | list[ICD]:
        """Retrieves ICDs. With no arguments, retrieves every ICD.

        Each filter field takes a value, a list of values, or an operator, and an ICD
        must match every field. A plain ``key`` returns one ICD.

        :param key: Keeps ICDs with this key.
        :param and_: Keeps ICDs that match every filter in the list.
        :param or_: Keeps ICDs that match any filter in the list.
        :param not_: Keeps ICDs that do not match the filter.
        :param where: A filter tree that the ICDs must also match.
        :param keys: Keeps ICDs with one of these keys.
        :param search_term: A fuzzy search term matched against ICD names.
        :param limit: The maximum number of ICDs to retrieve.
        :param after: Retrieves the ICDs past this position in key order: the key of
            the last ICD of the page before.
        :param one: Returns one ICD for any filter.
        :returns: One ICD for a plain ``key`` or when ``one`` is set, otherwise a list
            of ICDs. The list leaves out keys that do not exist.
        :raises NotFoundError: If one ICD is wanted and none matches.
        :raises MultipleFoundError: If one ICD is wanted and more than one matches.
        :raises ValidationError: If a filter field is None, or if one ICD is wanted
            beside a limit or a cursor.
        """
        single = returns_one(fields, one, limit=limit, after=after)
        flat = Where(key=None if keys is None else filter.Set[Key](in_=keys))
        req = RetrieveRequest(
            where=request_where(Where, filter.merge(where, flat), fields),
            search_term=search_term,
            limit=limit,
            after=after,
        )
        res = self._client.send("/icd/retrieve", req, RetrieveResponse)
        return sole("ICD", req, res.icds) if single else res.icds
